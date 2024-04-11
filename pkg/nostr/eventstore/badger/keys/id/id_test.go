package id

import (
	"bytes"
	"testing"

	"github.com/minio/sha256-simd"
	"lukechampine.com/frand"
	"mleku.dev/git/nostr/eventid"
	"mleku.dev/git/nostr/hex"
)

func TestT(t *testing.T) {
	fakeIdBytes := frand.Bytes(sha256.Size)
	fakeIdHex := hex.Enc(fakeIdBytes)
	id, err := eventid.New(fakeIdHex)
	if chk.E(err) {
		t.FailNow()
	}
	v := New(id)
	buf := new(bytes.Buffer)
	v.Write(buf)
	buf2 := bytes.NewBuffer(buf.Bytes())
	v2 := New("")
	el := v2.Read(buf2).(*T)
	if bytes.Compare(el.Val, v.Val) != 0 {
		t.Fatalf("expected %x got %x", v.Val, el.Val)
	}
}
