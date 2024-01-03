package nip19

import (
	"encoding/hex"
	"fmt"

	btcec "mleku.online/git/ec"
	log2 "mleku.online/git/log"

	"mleku.online/git/bech32"
	"mleku.online/git/ec/schnorr"
	secp "mleku.online/git/ec/secp"
)

var (
	log                    = log2.GetLogger()
	fails                  = log.D.Chk
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

const (
	// MinKeyStringLen is 56 because Bech32 needs 52 characters plus 4 for the HRP,
	// any string shorter than this cannot be a nostr key.
	MinKeyStringLen = 56
	HexKeyLen       = 64
	Bech32HRPLen    = 4
	SecHRP          = "nsec"
	PubHRP          = "npub"
	SigHRP          = "nsig"
)

// ConvertForBech32 performs the bit expansion required for encoding into
// Bech32.
func ConvertForBech32(b8 []byte) (b5 []byte, e error) {
	return bech32.ConvertBits(b8, 8, 5, true)
}

// ConvertFromBech32 collapses together the bit expanded 5 bit numbers encoded
// in bech32.
func ConvertFromBech32(b5 []byte) (b8 []byte, e error) {
	return bech32.ConvertBits(b5, 5, 8, true)
}

// SecretKeyToNsec encodes an secp256k1 secret key as a Bech32 string (nsec).
func SecretKeyToNsec(sk *secp.SecretKey) (encoded string, e error) {

	var b5 []byte
	if b5, e = ConvertForBech32(sk.Serialize()); e != nil {
		return
	}
	return bech32.Encode(SecHRP, b5)
}

// PublicKeyToNpub encodes a public kxey as a bech32 string (npub).
func PublicKeyToNpub(pk *secp.PublicKey) (encoded string, e error) {

	var bits5 []byte
	if bits5, e = ConvertForBech32(schnorr.SerializePubKey(pk)); e != nil {
		return
	}
	return bech32.Encode(PubHRP, bits5)
}

// NsecToSecretKey decodes a nostr secret key (nsec) and returns the secp256k1
// secret key.
func NsecToSecretKey(encoded string) (sk *secp.SecretKey, e error) {

	var b5, b8 []byte
	var hrp string
	hrp, b5, e = bech32.Decode(encoded)
	if e != nil {
		return
	}
	if hrp != SecHRP {
		e = fmt.Errorf("wrong human readable part, got '%s' want '%s'",
			hrp, SecHRP)
		return
	}
	b8, e = ConvertFromBech32(b5)
	if e != nil {
		return
	}
	sk = secp.SecKeyFromBytes(b8)
	return
}

// NpubToPublicKey decodes an nostr public key (npub) and returns an secp256k1
// public key.
func NpubToPublicKey(encoded string) (pk *secp.PublicKey, e error) {
	var b5, b8 []byte
	var hrp string
	hrp, b5, e = bech32.Decode(encoded)
	if e != nil {
		e = fmt.Errorf("ERROR: '%s'", e)
		return
	}
	if hrp != PubHRP {
		e = fmt.Errorf("wrong human readable part, got '%s' want '%s'",
			hrp, PubHRP)
		return
	}
	b8, e = ConvertFromBech32(b5)
	if e != nil {
		return
	}

	return schnorr.ParsePubKey(b8[:32])
}

// HexToPublicKey decodes a string that should be a 64 character long hex
// encoded public key into a btcec.PublicKey that can be used to verify a
// signature or encode to Bech32.
func HexToPublicKey(pk string) (p *btcec.PublicKey, e error) {
	if len(pk) != HexKeyLen {
		e = fmt.Errorf("seckey is %d bytes, must be %d", len(pk), HexKeyLen)
		return
	}
	var pb []byte
	if pb, e = hexDecode(pk); fails(e) {
		return
	}
	if p, e = schnorr.ParsePubKey(pb); fails(e) {
		return
	}
	return
}

// HexToSecretKey decodes a string that should be a 64 character long hex
// encoded public key into a btcec.PublicKey that can be used to verify a
// signature or encode to Bech32.
func HexToSecretKey(sk string) (s *btcec.SecretKey, e error) {
	if len(sk) != HexKeyLen {
		e = fmt.Errorf("seckey is %d bytes, must be %d", len(sk), HexKeyLen)
		return
	}
	var pb []byte
	if pb, e = hexDecode(sk); fails(e) {
		return
	}
	if s = secp.SecKeyFromBytes(pb); fails(e) {
		return
	}
	return
}

// EncodeSignature encodes a schnorr signature as Bech32 with the HRP "nsig" to
// be consistent with the key encodings 4 characters starting with 'n'.
func EncodeSignature(sig *schnorr.Signature) (str string, e error) {

	var b5 []byte
	b5, e = ConvertForBech32(sig.Serialize())
	if e != nil {
		e = fmt.Errorf("ERROR: '%s'", e)
		return
	}
	str, e = bech32.Encode(SigHRP, b5)
	return
}

// DecodeSignature decodes a Bech32 encoded nsig nostr (schnorr) signature into
// its runtime binary form.
func DecodeSignature(encoded string) (sig *schnorr.Signature, e error) {

	var b5, b8 []byte
	var hrp string
	hrp, b5, e = bech32.DecodeNoLimit(encoded)
	if e != nil {
		e = fmt.Errorf("ERROR: '%s'", e)
		return
	}
	if hrp != SigHRP {
		e = fmt.Errorf("wrong human readable part, got '%s' want '%s'",
			hrp, SigHRP)
		return
	}
	b8, e = ConvertFromBech32(b5)
	if e != nil {
		return
	}
	return schnorr.ParseSignature(b8[:64])
}

func GetPublicKey(sk string) (string, error) {
	b, err := hex.DecodeString(sk)
	if err != nil {
		return "", err
	}

	_, pk := btcec.PrivKeyFromBytes(b)
	return hex.EncodeToString(schnorr.SerializePubKey(pk)), nil
}
