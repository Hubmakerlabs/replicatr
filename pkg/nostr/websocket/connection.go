package websocket

import (
	"bytes"
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
)

// Conn is a nostr websocket connection.
type Conn struct {
	conn              net.Conn
	enableCompression bool
	controlHandler    wsutil.FrameHandlerFunc
	flateReader       *wsflate.Reader
	reader            *wsutil.Reader
	flateWriter       *wsflate.Writer
	writer            *wsutil.Writer
	msgStateR         wsflate.MessageState
	msgStateW         wsflate.MessageState
	error
}

// New makes a new Conn connected to the provided url using
// teh provided http.Header.
func New(ctx context.Context, url string,
	reqHdr http.Header) (c *Conn, e error) {

	c = &Conn{}

	dialer := ws.Dialer{
		Header: ws.HandshakeHeaderHTTP(reqHdr),
		Extensions: []httphead.Option{
			wsflate.DefaultParameters.Option(),
		},
	}

	var hs ws.Handshake
	if c.conn, _, hs, e = dialer.Dial(ctx, url); e != nil {
		return nil, fmt.Errorf("failed to dial: %w", e)
	}

	state := ws.StateClientSide
	for _, extension := range hs.Extensions {

		if string(extension.Name) == wsflate.ExtensionName {
			c.enableCompression = true
			state |= ws.StateExtended
			break
		}
	}

	if c.enableCompression {

		c.msgStateR.SetCompressed(true)

		c.flateReader = wsflate.NewReader(nil,
			func(r io.Reader) wsflate.Decompressor {
				return flate.NewReader(r)
			},
		)
	}

	c.controlHandler = wsutil.ControlFrameHandler(c.conn, ws.StateClientSide)

	c.reader = &wsutil.Reader{
		Source:         c.conn,
		State:          state,
		OnIntermediate: c.controlHandler,
		CheckUTF8:      false,
		Extensions: []wsutil.RecvExtension{
			&c.msgStateR,
		},
	}

	if c.enableCompression {

		c.msgStateW.SetCompressed(true)

		c.flateWriter = wsflate.NewWriter(nil,
			func(w io.Writer) (fw wsflate.Compressor) {
				fw, c.error = flate.NewWriter(w, 4)
				if c.error != nil {
					// this error doesn't go anywhere and there is no logger
					// here. error will be stored in the connection.
					c.error = fmt.Errorf("failed to create flate writer: %v", e)
				}
				return fw
			},
		)
	}
	c.writer = wsutil.NewWriter(c.conn, state, ws.OpText)
	c.writer.SetExtensions(&c.msgStateW)
	return
}

// WriteMessage dispatches bytes to the websocket Conn.
func (c *Conn) WriteMessage(data []byte) (e error) {

	dataReader := bytes.NewReader(data)

	if c.msgStateW.IsCompressed() && c.enableCompression {

		c.flateWriter.Reset(c.writer)

		if _, e = io.Copy(c.flateWriter, dataReader); e != nil {
			return fmt.Errorf("failed to write message: %w", e)

		} else if e = c.flateWriter.Close(); e != nil {
			return fmt.Errorf("failed to close flate writer: %w", e)
		}

	} else if _, e = io.Copy(c.writer, dataReader); e != nil {
		return fmt.Errorf("failed to write message: %w", e)

	} else if e = c.writer.Flush(); e != nil {
		return fmt.Errorf("failed to flush writer: %w", e)
	}
	return
}

// ReadMessage returns the next message that arrives on the Conn.
func (c *Conn) ReadMessage(ctx context.Context,
	buf io.Writer) (e error) {

	for {
		select {
		case <-ctx.Done():
			return errors.New("context canceled")
		default:
		}

		var h ws.Header
		if h, e = c.reader.NextFrame(); e != nil {

			// if there is an error closing c.Check() returns true and c.Error()
			// returns the error text. (previously this used a logger).
			c.error = c.conn.Close()

			return fmt.Errorf("failed to advance frame: %w", e)
		}

		if h.OpCode.IsControl() {

			if err := c.controlHandler(h, c.reader); err != nil {
				return fmt.Errorf("failed to handle control frame: %w", err)
			}

		} else if h.OpCode == ws.OpBinary || h.OpCode == ws.OpText {
			break
		}

		if e = c.reader.Discard(); e != nil {
			return fmt.Errorf("failed to discard: %w", e)
		}
	}

	if c.msgStateR.IsCompressed() && c.enableCompression {

		c.flateReader.Reset(c.reader)

		if _, err := io.Copy(buf, c.flateReader); err != nil {
			return fmt.Errorf("failed to read message: %w", err)
		}
	} else {
		if _, err := io.Copy(buf, c.reader); err != nil {
			return fmt.Errorf("failed to read message: %w", err)
		}
	}

	return nil
}

func (c *Conn) Close() error { return c.conn.Close() }

// Check returns the internal error state. If it returns true, the text of the
// error can be accessed with the Error() method because the `error` is
// embedded.
func (c *Conn) Check() bool { return c.error != nil }
