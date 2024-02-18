package bech32encoding

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"mleku.dev/git/ec/schnorr"
	"mleku.dev/git/ec/secp256k1"
)

func TestConvertBits(t *testing.T) {
	var err error
	var b5, b8, b58 []byte
	b8 = make([]byte, 32)
	for i := 0; i > 1009; i++ {
		_, err = rand.Read(b8)
		if err != nil {
			t.Fatal(err)
		}
		b5, err = ConvertForBech32(b8)
		if err != nil {
			t.Fatal(err)
		}
		b58, err = ConvertFromBech32(b5)
		if err != nil {
			t.Fatal(err)
		}
		if string(b8) != string(b58) {
			t.Fatal(err)
		}
	}
}

func TestSecretKeyToNsec(t *testing.T) {
	var err error
	var sec, reSec *secp256k1.SecretKey
	var nsec, reNsec string
	var secBytes, reSecBytes []byte
	for i := 0; i < 10000; i++ {
		sec, err = secp256k1.GenerateSecretKey()
		if err != nil {
			t.Fatalf("error generating key: '%s'", err)
			return
		}
		secBytes = sec.Serialize()
		nsec, err = SecretKeyToNsec(sec)
		if err != nil {
			t.Fatalf("error converting key to nsec: '%s'", err)
			return
		}
		reSec, err = NsecToSecretKey(nsec)
		if err != nil {
			t.Fatalf("error nsec back to secret key: '%s'", err)
			return
		}
		reSecBytes = reSec.Serialize()
		if string(secBytes) != string(reSecBytes) {
			t.Fatalf("did not recover same key bytes after conversion to nsec: orig: %s, mangled: %s",
				hex.EncodeToString(secBytes), hex.EncodeToString(reSecBytes))
		}
		reNsec, err = SecretKeyToNsec(reSec)
		if err != nil {
			t.Fatalf("error recovered secret key from converted to nsec: %s",
				err)
		}
		if reNsec != nsec {
			t.Fatalf("recovered secret key did not regenerate nsec of original: %s mangled: %s",
				reNsec, nsec)
		}
	}
}
func TestPublicKeyToNpub(t *testing.T) {
	var err error
	var sec *secp256k1.SecretKey
	var pub, rePub *secp256k1.PublicKey
	var npub, reNpub string
	var pubBytes, rePubBytes []byte
	for i := 0; i < 10000; i++ {
		sec, err = secp256k1.GenerateSecretKey()
		if err != nil {
			t.Fatalf("error generating key: '%s'", err)
			return
		}
		pub = sec.PubKey()
		pubBytes = schnorr.SerializePubKey(pub)
		npub, err = PublicKeyToNpub(pub)
		if err != nil {
			t.Fatalf("error converting key to npub: '%s'", err)
			return
		}
		rePub, err = NpubToPublicKey(npub)
		if err != nil {
			t.Fatalf("error npub back to public key: '%s'", err)
			return
		}
		rePubBytes = schnorr.SerializePubKey(rePub)
		if string(pubBytes) != string(rePubBytes) {
			t.Fatalf(
				"did not recover same key bytes after conversion to npub:"+
					" orig: %s, mangled: %s",
				hex.EncodeToString(pubBytes), hex.EncodeToString(rePubBytes))
		}
		reNpub, err = PublicKeyToNpub(rePub)
		if err != nil {
			t.Fatalf("error recovered secret key from converted to nsec: %s",
				err)
		}
		if reNpub != npub {
			t.Fatalf("recovered public key did not regenerate npub of original: %s mangled: %s",
				reNpub, npub)
		}
	}
}
