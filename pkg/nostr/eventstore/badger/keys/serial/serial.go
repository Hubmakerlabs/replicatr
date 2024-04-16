package serial

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

const Len = 8

// T is a badger DB serial number used for conflict free event record keys.
type T struct {
	Val []byte
}

var _ keys.Element = &T{}

// New returns a new serial record key.Element - if nil or short slice is given,
// initialize a fresh one with Len (for reading), otherwise if equal or longer,
// trim if long and store into struct (for writing).
func New(ser []byte) (p *T) {
	switch {
	case len(ser) < Len:
		log.I.Ln("empty serial")
		// allows use of nil to init
		ser = make([]byte, Len)
	default:
		ser = ser[:Len]
	}
	return &T{Val: ser}
}

// FromKey expects the last Len bytes of the given slice to be the serial.
func FromKey(k []byte) (p *T) {
	if len(k) < Len {
		panic("cannot get a serial without at least 8 bytes")
	}
	key := make([]byte, Len)
	copy(key, k[len(k)-Len:])
	return &T{Val: key}
}

func Make(s uint64) (ser []byte) {
	ser = make([]byte, 8)
	binary.BigEndian.PutUint64(ser, s)
	return
}

func (p *T) Write(buf *bytes.Buffer) {
	if len(p.Val) != Len {
		panic(fmt.Sprintln("must use New or initialize Val with len", Len))
	}
	buf.Write(p.Val)
}

func (p *T) Read(buf *bytes.Buffer) (el keys.Element) {
	// allow uninitialized struct
	if len(p.Val) != Len {
		p.Val = make([]byte, Len)
	}
	if n, err := buf.Read(p.Val); chk.E(err) || n != Len {
		return nil
	}
	return p
}

func (p *T) Len() int { return Len }
