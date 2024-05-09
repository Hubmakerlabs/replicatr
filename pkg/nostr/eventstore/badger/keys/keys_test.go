// package keys_test needs to be a different package name or the implementation
// types imports will circular
package keys_test

import (
	"bytes"
	"crypto/sha256"
	"os"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/kinder"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/pubkey"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"lukechampine.com/frand"
	"mleku.dev/git/ec/schnorr"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

func TestElement(t *testing.T) {
	var failed bool
	{ // construct a typical key type of structure
		// a prefix
		np := index.Version
		vp := index.New(byte(np))
		// an id
		fakeIdBytes := frand.Bytes(sha256.Size)
		fakeIdHex := hex.Enc(fakeIdBytes)
		i, err := eventid.New(fakeIdHex)
		if chk.E(err) {
			t.FailNow()
		}
		vid := id.New(i)
		// a kinder
		n := kind.T(1059)
		vk := kinder.New(n)
		// a pubkey
		fakePubkeyBytes := frand.Bytes(schnorr.PubKeyBytesLen)
		fakePubkeyHex := hex.Enc(fakePubkeyBytes)
		var vpk *pubkey.T
		vpk, err = pubkey.New(fakePubkeyHex)
		if chk.E(err) {
			t.FailNow()
		}
		// a createdat
		ts := timestamp.Now()
		vca := createdat.New(ts)
		// a serial
		fakeSerialBytes := frand.Bytes(serial.Len)
		vs := serial.New(fakeSerialBytes)
		// write Element list into buffer
		b := keys.Write(vp, vid, vk, vpk, vca, vs)
		// check that values decoded all correctly
		// we expect the following types, so we must create them:
		var vp2 = index.New(0)
		var vid2 = id.New("")
		var vk2 = kinder.New(0)
		var vpk2 *pubkey.T
		vpk2, err = pubkey.New()
		if chk.E(err) {
			t.FailNow()
		}
		var vca2 = createdat.New(0)
		var vs2 = serial.New(nil)
		// read it in
		keys.Read(b, vp2, vid2, vk2, vpk2, vca2, vs2)
		// this is a lot of tests, so use switch syntax
		switch {
		case bytes.Compare(vp.Val, vp2.Val) != 0:
			t.Logf("failed to decode correctly got %v expected %v", vp2.Val,
				vp.Val)
			failed = true
			fallthrough
		case bytes.Compare(vid.Val, vid2.Val) != 0:
			t.Logf("failed to decode correctly got %v expected %v", vid2.Val,
				vid.Val)
			failed = true
			fallthrough
		case vk.Val != vk2.Val:
			t.Logf("failed to decode correctly got %v expected %v", vk2.Val,
				vk.Val)
			failed = true
			fallthrough
		case bytes.Compare(vpk.Val, vpk2.Val) != 0:
			t.Logf("failed to decode correctly got %v expected %v", vpk2.Val,
				vpk.Val)
			failed = true
			fallthrough
		case vca.Val != vca2.Val:
			t.Logf("failed to decode correctly got %v expected %v", vca2.Val,
				vca.Val)
			failed = true
			fallthrough
		case bytes.Compare(vs.Val, vs2.Val) != 0:
			t.Logf("failed to decode correctly got %v expected %v", vpk2.Val,
				vpk.Val)
			failed = true
		}
	}
	{ // construct a counter value
		// a createdat
		ts := timestamp.Now()
		vca := createdat.New(ts)
		// write out values
		b := keys.Write(vca)
		// check that values decoded all correctly
		// we expect the following types, so we must create them:
		var vca2 = createdat.New(0)
		// read it in
		keys.Read(b, vca2)
		// check they match
		if vca.Val != vca2.Val {
			t.Logf("failed to decode correctly got %v expected %v", vca2.Val,
				vca.Val)
			failed = true
		}
	}
	if failed {
		t.FailNow()
	}
}
