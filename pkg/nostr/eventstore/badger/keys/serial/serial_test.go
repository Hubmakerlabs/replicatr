package serial_test

import (
	"bytes"
	"testing"

	"lukechampine.com/frand"
	"mleku.dev/git/nostr/eventstore/badger"
	"mleku.dev/git/nostr/eventstore/badger/keys/serial"
)

func TestT(t *testing.T) {
	fakeSerialBytes := frand.Bytes(badger.SerialLen)
	v := serial.New(fakeSerialBytes)
	buf := new(bytes.Buffer)
	v.Write(buf)
	buf2 := bytes.NewBuffer(buf.Bytes())
	v2 := &serial.T{} // or can use New(nil)
	el := v2.Read(buf2).(*serial.T)
	if bytes.Compare(el.Val, v.Val) != 0 {
		t.Fatalf("expected %x got %x", v.Val, el.Val)
	}
}
