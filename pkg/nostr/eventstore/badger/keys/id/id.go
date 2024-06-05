package id

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"mleku.net/slog"
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
	evid := string(evID[0])
	if len(evid) < 64 {
		evid = strings.Repeat("0", 64-len(evid)) + evid
	}
	if len(evid) > 64 {
		evid = evid[:64]
	}
	b, err := hex.Dec(evid[:Len*2])
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
