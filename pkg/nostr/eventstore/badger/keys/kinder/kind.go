package kinder

import (
	"bytes"
	"encoding/binary"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

const Len = 2

type T struct {
	Val kind.T
}

var _ keys.Element = &T{}

// New creates a new kinder.T for reading/writing kind.T values.
func New[V kind.T | uint16 | int](c V) (p *T) { return &T{Val: kind.T(c)} }

func Make[V kind.T | uint16 | int](c V) (v []byte) {
	v = make([]byte, Len)
	binary.BigEndian.PutUint16(v, uint16(c))
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
	v := binary.BigEndian.Uint16(b)
	c.Val = kind.T(v)
	return c
}

func (c *T) Len() int { return Len }
