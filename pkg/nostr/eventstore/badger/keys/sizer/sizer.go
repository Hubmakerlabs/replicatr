package sizer

import (
	"bytes"
	"encoding/binary"
	"os"

	"mleku.dev/git/nostr/eventstore/badger/keys"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

const Len = 4

type T struct {
	Val uint32
}

var _ keys.Element = &T{}

// New creates a new value with the underlying type uint32, that can be written
// to and read from a binary byte buffer.
//
// The reason why it is coded as generic with only one listed type is because it
// makes it possible to later on add another type, which is just a wrapper
// around uint32, that enables the use of other methods on it. This is because
// it became clear this was needed for the kinder.T type as well
func New[V uint32 | int](c V) (p *T) { return &T{Val: uint32(c)} }

func Make[V uint32 | int](c V) (v []byte) {
	v = make([]byte, Len)
	binary.BigEndian.PutUint32(v, uint32(c))
	return
}

func (c *T) Write(buf *bytes.Buffer) {
	buf.Write(Make(c.Val))
}

func (c *T) Read(buf *bytes.Buffer) (el keys.Element) {
	b := make([]byte, Len)
	if n, err := buf.Read(b); chk.E(err) || n != Len {
		return nil
	}
	c.Val = binary.BigEndian.Uint32(b)
	return c
}

func (c *T) Len() int { return Len }
