package event

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/object"
	"github.com/minio/sha256-simd"
	"mleku.dev/git/ec/schnorr"
	"mleku.dev/git/ec/secp256k1"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

func Hash(in []byte) (out []byte) {
	h := sha256.Sum256(in)
	return h[:]
}

// T is the primary datatype of nostr. This is the form of the structure
// that defines its JSON string based format.
type T struct {

	// ID is the SHA256 hash of the canonical encoding of the event
	ID eventid.T `json:"id"`

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

// Ascending is a slice of events that sorts in ascending chronological order
type Ascending []*T

func (ev Ascending) Len() int           { return len(ev) }
func (ev Ascending) Less(i, j int) bool { return ev[i].CreatedAt < ev[j].CreatedAt }
func (ev Ascending) Swap(i, j int)      { ev[i], ev[j] = ev[j], ev[i] }

// Descending sorts a slice of events in reverse chronological order (newest
// first)
type Descending []*T

func (e Descending) Len() int           { return len(e) }
func (e Descending) Less(i, j int) bool { return e[i].CreatedAt > e[j].CreatedAt }

func (e Descending) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

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

func (ev *T) MarshalJSON() (bytes []byte, err error) {
	return ev.ToObject().Bytes(), nil
}

func (ev *T) Serialize() []byte { return ev.ToObject().Bytes() }

// ToCanonical returns a structure that provides a byte stringer that generates
// the canonical form used to generate the ID hash that can be signed.
func (ev *T) ToCanonical() (o array.T) {
	return array.T{0, ev.PubKey, ev.CreatedAt, ev.Kind, ev.Tags, ev.Content}
}

// GetIDBytes returns the raw SHA256 hash of the canonical form of an T.
func (ev *T) GetIDBytes() []byte {
	canonical := ev.ToCanonical().Bytes()
	h := Hash(canonical)
	// log.T.Ln(string(canonical))
	return h
}

// GetID serializes and returns the event ID as a hexadecimal string.
func (ev *T) GetID() eventid.T {
	eid := eventid.T(hex.Enc(ev.GetIDBytes()))
	return eid
}

// CheckSignature checks if the signature is valid for the id (which is a hash
// of the serialized event content). returns an error if the signature itself is
// invalid.
func (ev *T) CheckSignature() (valid bool, err error) {

	// decode pubkey hex to bytes.
	var pkBytes []byte
	if pkBytes, err = hex.Dec(ev.PubKey); chk.D(err) {
		err = log.E.Err("event pubkey '%s' is invalid hex: %w", ev.PubKey, err)
		log.D.Ln(err)
		return
	}

	// parse pubkey bytes.
	var pk *secp256k1.PublicKey
	if pk, err = schnorr.ParsePubKey(pkBytes); chk.D(err) {
		err = log.E.Err("event has invalid pubkey '%s': %w", ev.PubKey, err)
		log.D.Ln(err)
		return
	}

	// decode signature hex to bytes.
	var sigBytes []byte
	if sigBytes, err = hex.Dec(ev.Sig); chk.D(err) {
		err = log.E.Err("signature '%s' is invalid hex: %w", ev.Sig, err)
		log.D.Ln(err)
		return
	}

	// parse signature bytes.
	var sig *schnorr.Signature
	if sig, err = schnorr.ParseSignature(sigBytes); chk.D(err) {
		err = log.E.Err("failed to parse signature: %w", err)
		log.D.Ln(err)
		return
	}

	// check signature.
	valid = sig.Verify(ev.GetIDBytes(), pk)
	return
}

// Sign signs an event with a given Secret Key encoded in hexadecimal.
func (ev *T) Sign(skStr string, so ...schnorr.SignOption) (err error) {

	// secret key hex must be 64 characters.
	if len(skStr) != 64 {
		err = log.E.Err("invalid secret key length, 64 required, got %d: %s",
			len(skStr), skStr)
		log.D.Ln(err)
		return
	}

	// decode secret key hex to bytes
	var skBytes []byte
	if skBytes, err = hex.Dec(skStr); chk.D(err) {
		err = log.E.Err("sign called with invalid secret key '%s': %w", skStr, err)
		log.D.Ln(err)
		return
	}

	// parse bytes to get secret key (size checks have been done).
	sk := secp256k1.SecKeyFromBytes(skBytes)
	ev.PubKey = hex.Enc(schnorr.SerializePubKey(sk.PubKey()))
	err = ev.SignWithSecKey(sk, so...)
	chk.D(err)
	return
}

// SignWithSecKey signs an event with a given *secp256xk1.SecretKey.
func (ev *T) SignWithSecKey(sk *secp256k1.SecretKey,
	so ...schnorr.SignOption) (err error) {

	// sign the event.
	var sig *schnorr.Signature
	id := ev.GetIDBytes()
	if sig, err = schnorr.Sign(sk, id, so...); chk.D(err) {
		return err
	}

	// we know ID is good so just coerce type.
	ev.ID = eventid.T(hex.Enc(id))

	// we know secret key is good so we can generate the public key.
	ev.PubKey = hex.Enc(schnorr.SerializePubKey(sk.PubKey()))
	log.D.Ln(ev.PubKey)
	ev.Sig = hex.Enc(sig.Serialize())
	log.D.Ln(ev.ToObject().String())
	return nil
}
