package id

import (
	"bytes"
	"fmt"
	"os"

	"mleku.dev/git/nostr/eventid"
	"mleku.dev/git/nostr/eventstore/badger/keys"
	"mleku.dev/git/nostr/hex"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

const Len = 8

type T struct {
	Val []byte
}

var _ keys.Element = &T{}

func New[V eventid.T | string](evID ...V) (p *T) {
	if len(evID) < 1 || len(evID[0]) < 1 {
		return &T{make([]byte, Len)}
	}
	b, err := hex.Dec(string(evID[0][:Len*2]))
	if chk.E(err) {
		return
	}
	return &T{Val: b}
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
