package connection

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
)

type C struct {
	Conn              net.Conn
	enableCompression bool
	controlHandler    wsutil.FrameHandlerFunc
	flateReader       *wsflate.Reader
	reader            *wsutil.Reader
	flateWriter       *wsflate.Writer
	writer            *wsutil.Writer
	msgStateR         *wsflate.MessageState
	msgStateW         *wsflate.MessageState
}

func NewConnection(c context.T, url string, requestHeader http.Header) (*C, error) {
	dialer := ws.Dialer{
		Header: ws.HandshakeHeaderHTTP(requestHeader),
		Extensions: []httphead.Option{
			wsflate.DefaultParameters.Option(),
		},
	}
	conn, _, hs, e := dialer.Dial(c, url)
	if e != nil {
		return nil, fmt.Errorf("failed to dial: %w", e)
	}

	enableCompression := false
	state := ws.StateClientSide
	for _, extension := range hs.Extensions {
		if string(extension.Name) == wsflate.ExtensionName {
			enableCompression = true
			state |= ws.StateExtended
			break
		}
	}

	// reader
	var flateReader *wsflate.Reader
	var msgStateR wsflate.MessageState
	if enableCompression {
		msgStateR.SetCompressed(true)

		flateReader = wsflate.NewReader(nil, func(r io.Reader) wsflate.Decompressor {
			return flate.NewReader(r)
		})
	}

	controlHandler := wsutil.ControlFrameHandler(conn, ws.StateClientSide)
	reader := &wsutil.Reader{
		Source:         conn,
		State:          state,
		OnIntermediate: controlHandler,
		CheckUTF8:      false,
		Extensions: []wsutil.RecvExtension{
			&msgStateR,
		},
	}

	// writer
	var flateWriter *wsflate.Writer
	var msgStateW wsflate.MessageState
	if enableCompression {
		msgStateW.SetCompressed(true)

		flateWriter = wsflate.NewWriter(nil, func(w io.Writer) wsflate.Compressor {
			fw, e := flate.NewWriter(w, 4)
			if e != nil {
				fmt.Printf("Failed to create flate writer: %v", e)
			}
			return fw
		})
	}

	writer := wsutil.NewWriter(conn, state, ws.OpText)
	writer.SetExtensions(&msgStateW)

	return &C{
		Conn:              conn,
		enableCompression: enableCompression,
		controlHandler:    controlHandler,
		flateReader:       flateReader,
		reader:            reader,
		msgStateR:         &msgStateR,
		flateWriter:       flateWriter,
		writer:            writer,
		msgStateW:         &msgStateW,
	}, nil
}

func (c *C) WriteMessage(data []byte) error {
	if c.msgStateW.IsCompressed() && c.enableCompression {
		c.flateWriter.Reset(c.writer)
		if _, e := io.Copy(c.flateWriter, bytes.NewReader(data)); e != nil {
			return fmt.Errorf("failed to write message: %w", e)
		}

		if e := c.flateWriter.Close(); e != nil {
			return fmt.Errorf("failed to close flate writer: %w", e)
		}
	} else {
		if _, e := io.Copy(c.writer, bytes.NewReader(data)); e != nil {
			return fmt.Errorf("failed to write message: %w", e)
		}
	}

	if e := c.writer.Flush(); e != nil {
		return fmt.Errorf("failed to flush writer: %w", e)
	}

	return nil
}

func (c *C) ReadMessage(cx context.T, buf io.Writer) error {
	for {
		select {
		case <-cx.Done():
			return errors.New("context canceled")
		default:
		}

		h, e := c.reader.NextFrame()
		if e != nil {
			c.Conn.Close()
			return fmt.Errorf("failed to advance frame: %w", e)
		}

		if h.OpCode.IsControl() {
			if e := c.controlHandler(h, c.reader); e != nil {
				return fmt.Errorf("failed to handle control frame: %w", e)
			}
		} else if h.OpCode == ws.OpBinary ||
			h.OpCode == ws.OpText {
			break
		}

		if e := c.reader.Discard(); e != nil {
			return fmt.Errorf("failed to discard: %w", e)
		}
	}

	if c.msgStateR.IsCompressed() && c.enableCompression {
		c.flateReader.Reset(c.reader)
		if _, e := io.Copy(buf, c.flateReader); e != nil {
			return fmt.Errorf("failed to read message: %w", e)
		}
	} else {
		if _, e := io.Copy(buf, c.reader); e != nil {
			return fmt.Errorf("failed to read message: %w", e)
		}
	}

	return nil
}

func (c *C) Close() error {
	return c.Conn.Close()
}
