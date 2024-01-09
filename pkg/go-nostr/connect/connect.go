package connect

import (
	"bytes"
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
)

var log = log2.GetStd()

type Connection struct {
	Conn              net.Conn
	enableCompression bool
	controlHandler    wsutil.FrameHandlerFunc
	flateReader       *wsflate.Reader
	reader            *wsutil.Reader
	flateWriter       *wsflate.Writer
	writer            *wsutil.Writer
	msgState          *wsflate.MessageState
}

func NewConnection(ctx context.Context, url string, requestHeader http.Header) (*Connection, error) {
	dialer := ws.Dialer{
		Header: ws.HandshakeHeaderHTTP(requestHeader),
		Extensions: []httphead.Option{
			wsflate.DefaultParameters.Option(),
		},
	}
	conn, _, hs, e := dialer.Dial(ctx, url)
	if log.Fail(e) {
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
	var msgState wsflate.MessageState
	if enableCompression {
		msgState.SetCompressed(true)

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
			&msgState,
		},
	}

	// writer
	var flateWriter *wsflate.Writer
	if enableCompression {
		flateWriter = wsflate.NewWriter(nil, func(w io.Writer) wsflate.Compressor {
			fw, e := flate.NewWriter(w, 4)
			if log.Fail(e) {
				log.E.F("Failed to create flate writer: %v", e)
			}
			return fw
		})
	}

	writer := wsutil.NewWriter(conn, state, ws.OpText)
	writer.SetExtensions(&msgState)

	return &Connection{
		Conn:              conn,
		enableCompression: enableCompression,
		controlHandler:    controlHandler,
		flateReader:       flateReader,
		reader:            reader,
		flateWriter:       flateWriter,
		msgState:          &msgState,
		writer:            writer,
	}, nil
}

func (c *Connection) WriteMessage(data []byte) (e error) {
	if c.msgState.IsCompressed() && c.enableCompression {
		c.flateWriter.Reset(c.writer)
		if _, e := io.Copy(c.flateWriter, bytes.NewReader(data)); log.Fail(e) {
			return fmt.Errorf("failed to write message: %w", e)
		}

		if e := c.flateWriter.Close(); log.Fail(e) {
			return fmt.Errorf("failed to close flate writer: %w", e)
		}
	} else {
		if _, e := io.Copy(c.writer, bytes.NewReader(data)); log.Fail(e) {
			return fmt.Errorf("failed to write message: %w", e)
		}
	}

	if e := c.writer.Flush(); log.Fail(e) {
		return fmt.Errorf("failed to flush writer: %w", e)
	}

	return nil
}

func (c *Connection) ReadMessage(ctx context.Context, buf io.Writer) (e error) {
	for {
		select {
		case <-ctx.Done():
			return errors.New("context canceled")
		default:
		}

		h, e := c.reader.NextFrame()
		if log.Fail(e) {
			c.Conn.Close()
			return fmt.Errorf("failed to advance frame: %w", e)
		}

		if h.OpCode.IsControl() {
			if e = c.controlHandler(h, c.reader); log.Fail(e) {
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

	if c.msgState.IsCompressed() && c.enableCompression {
		c.flateReader.Reset(c.reader)
		if _, e := io.Copy(buf, c.flateReader); log.Fail(e) {
			return fmt.Errorf("failed to read message: %w", e)
		}
	} else {
		if _, e := io.Copy(buf, c.reader); e != nil {
			return fmt.Errorf("failed to read message: %w", e)
		}
	}

	return nil
}

func (c *Connection) Close() (e error) {
	return c.Conn.Close()
}
