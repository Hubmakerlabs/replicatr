package createdat

import (
	"bytes"
	"testing"

	"mleku.dev/git/nostr/timestamp"
)

func TestT(t *testing.T) {
	n := timestamp.Now()
	v := New(n)
	buf := new(bytes.Buffer)
	v.Write(buf)
	buf2 := bytes.NewBuffer(buf.Bytes())
	v2 := New(0)
	el := v2.Read(buf2).(*T)
	if el.Val != n {
		t.Fatalf("expected %d got %d", n.Int(), el.Val.Int())
	}
}
