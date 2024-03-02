package acl

import (
	"testing"

	"lukechampine.com/frand"
	"mleku.dev/git/ec/schnorr"
	"mleku.dev/git/ec/secp256k1"
	"mleku.dev/git/nostr/hex"
	"mleku.dev/git/nostr/timestamp"
)

var testRelaySec = "f16dca5c36931305a4ac30d31b77962af96ea6b7240736da11af318fb7e11317"

func TestT(t *testing.T) {
	// generate a bunch of deterministic random pubkeys
	// might as well use the test relay pubkey
	seed, err := hex.Dec(testRelaySec)
	if err != nil {
		t.Fatal(err)
	}
	src := frand.NewCustom(seed, 128, 20)
	var pubkeys []string
	var sec *secp256k1.SecretKey
	for i := 0; i < 10; i++ {
		if sec, err = secp256k1.GenerateSecretKeyFromRand(src); err != nil {
			t.Fatal(err)
		}
		pub := sec.PubKey()
		pubBytes := schnorr.SerializePubKey(pub)
		pubkeys = append(pubkeys, hex.Enc(pubBytes))
	}
	aclT := &T{}
	for i := range pubkeys {
		role := (i % (len(RoleStrings) - 1)) + 1
		en := &Entry{
			Role:         Role(role),
			Pubkey:       pubkeys[i],
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
	frand.Shuffle(len(pubkeys), func(i, j int) {
		pubkeys[i], pubkeys[j] = pubkeys[j], pubkeys[i]
	})
	for i := range pubkeys {
		if err = aclT.DeleteEntry(pubkeys[i]); err != nil {
			t.Fatal(err)
		}
	}
}
