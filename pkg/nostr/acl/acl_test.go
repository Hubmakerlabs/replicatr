package acl

import (
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/ec/schnorr"
	"github.com/Hubmakerlabs/replicatr/pkg/ec/secp256k1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"lukechampine.com/frand"
)

var testRelaySec = "f16dca5c36931305a4ac30d31b77962af96ea6b7240736da11af318fb7e11317"

func TestT(t *testing.T) {
	// generate a bunch of deterministic random pub keys might as well use the test
	// relay pubkey
	seed, err := hex.Dec(testRelaySec)
	if err != nil {
		t.Fatal(err)
	}
	src := frand.NewCustom(seed, 128, 20)
	var pubKeys []string
	var sec *secp256k1.SecretKey
	for i := 0; i < 10; i++ {
		if sec, err = secp256k1.GenerateSecretKeyFromRand(src); err != nil {
			t.Fatal(err)
		}
		pub := sec.PubKey()
		pubBytes := schnorr.SerializePubKey(pub)
		pubKeys = append(pubKeys, hex.Enc(pubBytes))
	}
	aclT := &T{}
	for i := range pubKeys {
		role := (i % (len(RoleStrings) - 1)) + 1
		en := &Entry{
			Role:         Role(role),
			Pubkey:       pubKeys[i],
			Created:      timestamp.Now() - 1,
			LastModified: timestamp.Now(),
			Expires:      timestamp.Now() + 100000,
		}
		if err = aclT.AddEntry(en); err != nil {
			t.Fatal(err)
		}
		ev := en.ToEvent()
		if err = ev.Sign(testRelaySec); err != nil {
			t.Fatal(err)
		}
		var e *Entry
		if e, err = aclT.FromEvent(ev); err != nil {
			t.Fatal(err)
		}
		_ = e
	}
	frand.Shuffle(len(pubKeys), func(i, j int) {
		pubKeys[i], pubKeys[j] = pubKeys[j], pubKeys[i]
	})
	for i := range pubKeys {
		if err = aclT.DeleteEntry(pubKeys[i]); err != nil {
			t.Fatal(err)
		}
	}
}
