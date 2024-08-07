package createdat

import (
	"bytes"
	"encoding/binary"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

var log, chk = slog.New(os.Stderr)

const Len = 8

type T struct {
	Val timestamp.T
}

var _ keys.Element = &T{}

func New(c timestamp.T) (p *T) { return &T{Val: c} }

func (c *T) Write(buf *bytes.Buffer) {
	buf.Write(c.Val.Bytes())
}

func (c *T) Read(buf *bytes.Buffer) (el keys.Element) {
	b := make([]byte, Len)
	if n, err := buf.Read(b); chk.E(err) || n != Len {
		return nil
	}
	c.Val = timestamp.FromUnix(int64(binary.BigEndian.Uint64(b)))
	return c
}

func (c *T) Len() int { return Len }

// FromKey expects to find a datestamp in the 8 bytes before a serial in a key.
func FromKey(k []byte) (p *T) {
	if len(k) < Len+serial.Len {
		panic("cannot get a serial without at least 16 bytes")
	}
	key := make([]byte, Len)
	copy(key, k[len(k)-serial.Len:len(k)-Len+serial.Len])
	return &T{Val: timestamp.FromBytes(key)}
}
