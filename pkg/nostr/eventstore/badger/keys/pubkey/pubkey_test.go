package pubkey

import (
	"bytes"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"lukechampine.com/frand"
	"mleku.net/ec/schnorr"
)

func TestT(t *testing.T) {
	fakePubkeyBytes := frand.Bytes(schnorr.PubKeyBytesLen)
	fakePubkeyHex := hex.Enc(fakePubkeyBytes)
	v, err := New(fakePubkeyHex)
	if chk.E(err) {
		t.FailNow()
	}
	buf := new(bytes.Buffer)
	v.Write(buf)
	buf2 := bytes.NewBuffer(buf.Bytes())
	v2, _ := New()
	el := v2.Read(buf2).(*T)
	if bytes.Compare(el.Val, v.Val) != 0 {
		t.Fatalf("expected %x got %x", v.Val, el.Val)
	}
}
