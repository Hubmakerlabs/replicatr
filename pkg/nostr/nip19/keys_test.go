package nip19

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/minio/sha256-simd"
	"github.com/Hubmakerlabs/replicatr/pkg/ec/schnorr"
	secp256k1 "github.com/Hubmakerlabs/replicatr/pkg/ec/secp"
)

func TestConvertBits(t *testing.T) {
	var e error
	var b5, b8, b58 []byte
	b8 = make([]byte, 32)
	for i := 0; i > 1009; i++ {
		_, e = rand.Read(b8)
		if e != nil {
			t.Fatal(e)
		}
		b5, e = ConvertForBech32(b8)
		if e != nil {
			t.Fatal(e)
		}
		b58, e = ConvertFromBech32(b5)
		if e != nil {
			t.Fatal(e)
		}
		if string(b8) != string(b58) {
			t.Fatal(e)
		}
	}
}

func TestSecretKeyToNsec(t *testing.T) {
	var e error
	var sec, reSec *secp256k1.SecretKey
	var nsec, reNsec string
	var secBytes, reSecBytes []byte
	for i := 0; i < 10000; i++ {
		sec, e = secp256k1.GenerateSecretKey()
		if e != nil {
			t.Fatalf("error generating key: '%s'", e)
			return
		}
		secBytes = sec.Serialize()
		nsec, e = SecretKeyToNsec(sec)
		if e != nil {
			t.Fatalf("error converting key to nsec: '%s'", e)
			return
		}
		reSec, e = NsecToSecretKey(nsec)
		if e != nil {
			t.Fatalf("error nsec back to secret key: '%s'", e)
			return
		}
		reSecBytes = reSec.Serialize()
		if string(secBytes) != string(reSecBytes) {
			t.Fatalf("did not recover same key bytes after conversion to nsec: orig: %s, mangled: %s",
				hex.EncodeToString(secBytes), hex.EncodeToString(reSecBytes))
		}
		reNsec, e = SecretKeyToNsec(reSec)
		if e != nil {
			t.Fatalf("error recovered secret key from converted to nsec: %s",
				e)
		}
		if reNsec != nsec {
			t.Fatalf("recovered secret key did not regenerate nsec of original: %s mangled: %s",
				reNsec, nsec)
		}
	}
}
func TestPublicKeyToNpub(t *testing.T) {
	var e error
	var sec *secp256k1.SecretKey
	var pub, rePub *secp256k1.PublicKey
	var npub, reNpub string
	var pubBytes, rePubBytes []byte
	for i := 0; i < 10000; i++ {
		sec, e = secp256k1.GenerateSecretKey()
		if e != nil {
			t.Fatalf("error generating key: '%s'", e)
			return
		}
		pub = sec.PubKey()
		pubBytes = schnorr.SerializePubKey(pub)
		npub, e = PublicKeyToNpub(pub)
		if e != nil {
			t.Fatalf("error converting key to npub: '%s'", e)
			return
		}
		rePub, e = NpubToPublicKey(npub)
		if e != nil {
			t.Fatalf("error npub back to public key: '%s'", e)
			return
		}
		rePubBytes = schnorr.SerializePubKey(rePub)
		if string(pubBytes) != string(rePubBytes) {
			t.Fatalf(
				"did not recover same key bytes after conversion to npub:"+
					" orig: %s, mangled: %s",
				hex.EncodeToString(pubBytes), hex.EncodeToString(rePubBytes))
		}
		reNpub, e = PublicKeyToNpub(rePub)
		if e != nil {
			t.Fatalf("error recovered secret key from converted to nsec: %s",
				e)
		}
		if reNpub != npub {
			t.Fatalf("recovered public key did not regenerate npub of original: %s mangled: %s",
				reNpub, npub)
		}
	}
}

func TestSignatures(t *testing.T) {
	var e error
	var sec *secp256k1.SecretKey
	var pub *secp256k1.PublicKey
	bytes := make([]byte, 256)
	hashed := make([]byte, 32)
	var sig, deSig *schnorr.Signature
	var nsig string
	for i := 0; i < 10000; i++ {
		sec, e = secp256k1.GenerateSecretKey()
		if e != nil {
			t.Fatalf("error generating key: '%s'", e)
			return
		}
		pub = sec.PubKey()
		_, e = rand.Read(bytes)
		if e != nil {
			t.Fatal(e)
		}
		hashArray := sha256.Sum256(bytes)
		copy(hashed, hashArray[:])
		sig, e = schnorr.Sign(sec, hashed)
		if e != nil {
			t.Fatal(e)
		}
		nsig, e = EncodeSignature(sig)
		if e != nil {
			t.Fatal(e)
		}
		deSig, e = DecodeSignature(nsig)
		if e != nil {
			t.Fatal(e)
		}
		if !deSig.Verify(hashed, pub) {
			t.Fatal("signature failed but should not have failed")
		}
	}
}
