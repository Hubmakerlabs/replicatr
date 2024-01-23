package bech32encoding

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/bech32"
	"github.com/Hubmakerlabs/replicatr/pkg/ec"
	"github.com/Hubmakerlabs/replicatr/pkg/ec/schnorr"
	"github.com/Hubmakerlabs/replicatr/pkg/ec/secp256k1"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

var log = slog.GetStd()

const (
	// MinKeyStringLen is 56 because Bech32 needs 52 characters plus 4 for the HRP,
	// any string shorter than this cannot be a nostr key.
	MinKeyStringLen = 56
	HexKeyLen       = 64
	Bech32HRPLen    = 4
	SecHRP          = "nsec"
	PubHRP          = "npub"
)

// ConvertForBech32 performs the bit expansion required for encoding into
// Bech32.
func ConvertForBech32(b8 []byte) (b5 []byte, err error) {
	return bech32.ConvertBits(b8, 8, 5, true)
}

// ConvertFromBech32 collapses together the bit expanded 5 bit numbers encoded
// in bech32.
func ConvertFromBech32(b5 []byte) (b8 []byte, err error) {
	return bech32.ConvertBits(b5, 5, 8, true)
}

// SecretKeyToNsec encodes an secp256k1 secret key as a Bech32 string (nsec).
func SecretKeyToNsec(sk *secp256k1.SecretKey) (encoded string, err error) {

	var b5 []byte
	if b5, err = ConvertForBech32(sk.Serialize()); err != nil {
		return
	}
	return bech32.Encode(SecHRP, b5)
}

// PublicKeyToNpub encodes a public kxey as a bech32 string (npub).
func PublicKeyToNpub(pk *secp256k1.PublicKey) (encoded string, err error) {

	var bits5 []byte
	if bits5, err = ConvertForBech32(schnorr.SerializePubKey(pk)); err != nil {
		return
	}
	return bech32.Encode(PubHRP, bits5)
}

// NsecToSecretKey decodes a nostr secret key (nsec) and returns the secp256k1
// secret key.
func NsecToSecretKey(encoded string) (sk *secp256k1.SecretKey, err error) {

	var b5, b8 []byte
	var hrp string
	hrp, b5, err = bech32.Decode(encoded)
	if err != nil {
		return
	}
	if hrp != SecHRP {
		err = fmt.Errorf("wrong human readable part, got '%s' want '%s'",
			hrp, SecHRP)
		return
	}
	b8, err = ConvertFromBech32(b5)
	if err != nil {
		return
	}
	sk = secp256k1.SecKeyFromBytes(b8)
	return
}

// NpubToPublicKey decodes an nostr public key (npub) and returns an secp256k1
// public key.
func NpubToPublicKey(encoded string) (pk *secp256k1.PublicKey, err error) {
	var b5, b8 []byte
	var hrp string
	hrp, b5, err = bech32.Decode(encoded)
	if err != nil {
		err = fmt.Errorf("ERROR: '%s'", err)
		return
	}
	if hrp != PubHRP {
		err = fmt.Errorf("wrong human readable part, got '%s' want '%s'",
			hrp, PubHRP)
		return
	}
	b8, err = ConvertFromBech32(b5)
	if err != nil {
		return
	}

	return schnorr.ParsePubKey(b8[:32])
}

// HexToPublicKey decodes a string that should be a 64 character long hex
// encoded public key into a btcec.PublicKey that can be used to verify a
// signature or encode to Bech32.
func HexToPublicKey(pk string) (p *btcec.PublicKey, err error) {
	if len(pk) != HexKeyLen {
		err = fmt.Errorf("seckey is %d bytes, must be %d", len(pk), HexKeyLen)
		return
	}
	var pb []byte
	if pb, err = hex.Dec(pk); log.Fail(err) {
		return
	}
	if p, err = schnorr.ParsePubKey(pb); log.Fail(err) {
		return
	}
	return
}

// HexToSecretKey decodes a string that should be a 64 character long hex
// encoded public key into a btcec.PublicKey that can be used to verify a
// signature or encode to Bech32.
func HexToSecretKey(sk string) (s *btcec.SecretKey, err error) {
	if len(sk) != HexKeyLen {
		err = fmt.Errorf("seckey is %d bytes, must be %d", len(sk), HexKeyLen)
		return
	}
	var pb []byte
	if pb, err = hex.Dec(sk); log.Fail(err) {
		return
	}
	if s = secp256k1.SecKeyFromBytes(pb); log.Fail(err) {
		return
	}
	return
}

func GetPublicKey(sk string) (s string, err error) {
	var b []byte
	b, err = hex.Dec(sk)
	if err != nil {
		return "", err
	}

	_, pk := btcec.PrivKeyFromBytes(b)
	return hex.Enc(schnorr.SerializePubKey(pk)), nil
}
