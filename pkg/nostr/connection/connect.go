package connection

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type C struct {
	Conn              net.Conn
	enableCompression bool
	controlHandler    wsutil.FrameHandlerFunc
	flateReader       *wsflate.Reader
	reader            *wsutil.Reader
	flateWriter       *wsflate.Writer
	writer            *wsutil.Writer
	msgState          *wsflate.MessageState
}

func NewConnection(c context.T, url string, requestHeader http.Header) (connection *C, err error) {
	dialer := ws.Dialer{
		Header: ws.HandshakeHeaderHTTP(requestHeader),
		Extensions: []httphead.Option{
			wsflate.DefaultParameters.Option(),
		},
	}
	conn, _, hs, err := dialer.Dial(c, url)
	if chk.D(err) {
		return nil, fmt.Errorf("failed to dial: %w", err)
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
			var fw *flate.Writer
			fw, err = flate.NewWriter(w, 4)
			if chk.D(err) {
				log.E.F("Failed to create flate writer: %v", err)
			}
			return fw
		})
	}
	writer := wsutil.NewWriterSize(conn, state, ws.OpText, app.MaxMessageSize)
	writer.SetExtensions(&msgState)
	connection = &C{
		Conn:              conn,
		enableCompression: enableCompression,
		controlHandler:    controlHandler,
		flateReader:       flateReader,
		reader:            reader,
		flateWriter:       flateWriter,
		msgState:          &msgState,
		writer:            writer,
	}
	return
}

func (c *C) WriteMessage(data []byte) (err error) {
	if c.msgState.IsCompressed() && c.enableCompression {
		c.flateWriter.Reset(c.writer)
		if _, err = io.Copy(c.flateWriter, bytes.NewReader(data)); chk.D(err) {
			return fmt.Errorf("failed to write message: %w", err)
		}
		if err = c.flateWriter.Close(); chk.D(err) {
			return fmt.Errorf("failed to close flate writer: %w", err)
		}
	} else {
		if _, err = io.Copy(c.writer, bytes.NewReader(data)); chk.D(err) {
			return fmt.Errorf("failed to write message: %w", err)
		}
	}
	if err = c.writer.Flush(); chk.D(err) {
		return fmt.Errorf("failed to flush writer: %w", err)
	}
	return nil
}

func (c *C) ReadMessage(cx context.T, buf io.Writer) (err error) {
	for {
		select {
		case <-cx.Done():
			return errors.New("context canceled")
		default:
		}
		var h ws.Header
		h, err = c.reader.NextFrame()
		if chk.D(err) {
			chk.D(c.Conn.Close())
			return fmt.Errorf("failed to advance frame: %w", err)
		}
		if h.OpCode.IsControl() {
			if err = c.controlHandler(h, c.reader); chk.D(err) {
				return fmt.Errorf("failed to handle control frame: %w", err)
			}
		} else if h.OpCode == ws.OpBinary ||
			h.OpCode == ws.OpText {
			break
		}
		if err = c.reader.Discard(); chk.E(err) {
			return fmt.Errorf("failed to discard: %w", err)
		}
	}
	if c.msgState.IsCompressed() && c.enableCompression {
		c.flateReader.Reset(c.reader)
		if _, err = io.Copy(buf, c.flateReader); chk.D(err) {
			return fmt.Errorf("failed to read message: %w", err)
		}
	} else {
		if _, err = io.Copy(buf, c.reader); chk.E(err) {
			return fmt.Errorf("failed to read message: %w", err)
		}
	}
	return nil
}

func (c *C) Close() (err error) {
	return c.Conn.Close()
}
