package event

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/ec/schnorr"
	secp256k1 "github.com/Hubmakerlabs/replicatr/pkg/ec/secp"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/object"
	"github.com/minio/sha256-simd"
)

var log = log2.GetStd()

func Hash(in []byte) (out []byte) {
	h := sha256.Sum256(in)
	return h[:]
}

// T is the primary datatype of nostr. This is the form of the structure
// that defines its JSON string based format.
type T struct {

	// ID is the SHA256 hash of the canonical encoding of the event
	ID eventid.EventID `json:"id"`

	// PubKey is the public key of the event creator in *hexadecimal* format
	PubKey string `json:"pubkey"`

	// CreatedAt is the UNIX timestamp of the event according to the event
	// creator (never trust a timestamp!)
	CreatedAt timestamp.T `json:"created_at"`

	// Kind is the nostr protocol code for the type of event. See kind.T
	Kind kind.T `json:"kind"`

	// Tags are a list of tags, which are a list of strings usually structured
	// as a 3 layer scheme indicating specific features of an event.
	Tags tags.T `json:"tags"`

	// Content is an arbitrary string that can contain anything, but usually
	// conforming to a specification relating to the Kind and the Tags.
	Content string `json:"content"`

	// Sig is the signature on the ID hash that validates as coming from the
	// Pubkey.
	Sig string `json:"sig"`
}

func (ev *T) ToObject() (o object.T) {
	return object.T{
		{"id", ev.ID},
		{"pubkey", ev.PubKey},
		{"created_at", ev.CreatedAt},
		{"kind", ev.Kind},
		{"tags", ev.Tags},
		{"content", ev.Content},
		{"sig", ev.Sig},
	}
}

func (ev *T) MarshalJSON() (bytes []byte, e error) {
	return ev.ToObject().Bytes(), nil
}

func (ev *T) Serialize() []byte { return ev.ToObject().Bytes() }

// ToCanonical returns a structure that provides a byte stringer that generates
// the canonical form used to generate the ID hash that can be signed.
func (ev *T) ToCanonical() (o array.T) {
	// log.D.S(ev)
	return array.T{0, ev.PubKey, ev.CreatedAt, ev.Kind, ev.Tags, ev.Content}
}

// GetIDBytes returns the raw SHA256 hash of the canonical form of an T.
func (ev *T) GetIDBytes() []byte { return Hash(ev.ToCanonical().Bytes()) }

// GetID serializes and returns the event ID as a hexadecimal string.
func (ev *T) GetID() eventid.EventID { return eventid.EventID(hex.Enc(ev.GetIDBytes())) }

// CheckSignature checks if the signature is valid for the id (which is a hash
// of the serialized event content). returns an error if the signature itself is
// invalid.
func (ev *T) CheckSignature() (valid bool, e error) {

	// decode pubkey hex to bytes.
	var pkBytes []byte
	if pkBytes, e = hex.Dec(ev.PubKey); log.Fail(e) {
		e = fmt.Errorf("event pubkey '%s' is invalid hex: %w", ev.PubKey, e)
		return
	}

	// parse pubkey bytes.
	var pk *secp256k1.PublicKey
	if pk, e = schnorr.ParsePubKey(pkBytes); log.Fail(e) {
		e = fmt.Errorf("event has invalid pubkey '%s': %w", ev.PubKey, e)
		return
	}

	// decode signature hex to bytes.
	var sigBytes []byte
	if sigBytes, e = hex.Dec(ev.Sig); log.Fail(e) {
		e = fmt.Errorf("signature '%s' is invalid hex: %w", ev.Sig, e)
		return
	}

	// parse signature bytes.
	var sig *schnorr.Signature
	if sig, e = schnorr.ParseSignature(sigBytes); log.Fail(e) {
		e = fmt.Errorf("failed to parse signature: %w", e)
		return
	}

	// check signature.
	valid = sig.Verify(ev.GetIDBytes(), pk)
	return
}

// Sign signs an event with a given Secret Key encoded in hexadecimal.
func (ev *T) Sign(skStr string, so ...schnorr.SignOption) (e error) {

	// secret key hex must be 64 characters.
	if len(skStr) != 64 {
		return fmt.Errorf("invalid secret key length, 64 required, got %d: %s",
			len(skStr), skStr)
	}

	// decode secret key hex to bytes
	var skBytes []byte
	if skBytes, e = hex.Dec(skStr); log.Fail(e) {
		return fmt.Errorf("sign called with invalid secret key '%s': %w",
			skStr, e)
	}

	// parse bytes to get secret key (size checks have been done).
	sk := secp256k1.SecKeyFromBytes(skBytes)

	return ev.SignWithSecKey(sk, so...)
}

// SignWithSecKey signs an event with a given *secp256xk1.SecretKey.
func (ev *T) SignWithSecKey(sk *secp256k1.SecretKey,
	so ...schnorr.SignOption) (e error) {

	// sign the event.
	var sig *schnorr.Signature
	id := ev.GetIDBytes()
	if sig, e = schnorr.Sign(sk, id, so...); log.Fail(e) {
		return e
	}

	// we know ID is good so just coerce type.
	ev.ID = eventid.EventID(hex.Enc(id))

	// we know secret key is good so we can generate the public key.
	ev.PubKey = hex.Enc(schnorr.SerializePubKey(sk.PubKey()))
	ev.Sig = hex.Enc(sig.Serialize())
	return nil
}
