package kinder

import (
	"bytes"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
)

func TestT(t *testing.T) {
	n := kind.T(1059)
	v := New(n)
	buf := new(bytes.Buffer)
	v.Write(buf)
	buf2 := bytes.NewBuffer(buf.Bytes())
	v2 := New(0)
	el := v2.Read(buf2).(*T)
	if el.Val != n {
		t.Fatalf("expected %d got %d", n, el.Val)
	}
}
