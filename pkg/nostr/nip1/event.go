package nip1

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	secp256k1 "mleku.online/git/ec/secp"
	log2 "mleku.online/git/log"
	"mleku.online/git/replicatr/pkg/jsontext"
	"mleku.online/git/replicatr/pkg/nostr/kind"
	"mleku.online/git/replicatr/pkg/nostr/tags"
	"mleku.online/git/replicatr/pkg/nostr/timestamp"

	"github.com/minio/sha256-simd"
	ec "mleku.online/git/ec"
	"mleku.online/git/ec/schnorr"
)

var (
	log   = log2.GetLogger()
	fails = log.E.Chk
)

// Event is the primary datatype of nostr. This is the form of the structure
// that defines its JSON string based format.
type Event struct {

	// ID is the SHA256 hash of the canonical encoding of the event
	ID string `json:"id"`

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

// Event Stringer interface, just returns the raw JSON as a string.
func (ev *Event) String() (s string) {
	if j, e := json.Marshal(ev); !fails(e) {
		s = string(j)
	}
	return
}

// GetID serializes and returns the event ID as a string.
func (ev *Event) GetID() string {
	h := sha256.Sum256(ev.Serialize())
	return hex.EncodeToString(h[:])
}

// Serialize outputs a byte array that can be hashed/signed to
// identify/authenticate. JSON encoding as defined in RFC4627.
func (ev *Event) Serialize() []byte {

	// the serialization process is just putting everything into a JSON array so
	// the order is kept. See NIP-01
	dst := make([]byte, 0)

	// the header portion is easy to serialize [0,"pubkey",created_at,kind,[
	dst = append(dst, []byte(
		fmt.Sprintf(
			"[0,\"%s\",%d,%d,",
			ev.PubKey,
			ev.CreatedAt,
			ev.Kind,
		))...)

	// tags
	dst = ev.Tags.MarshalTo(dst)
	dst = append(dst, ',')

	// content needs to be escaped in general as it is user generated.
	dst = append(dst, jsontext.EscapeJSONStringAndWrap(ev.Content)...)
	dst = append(dst, ']')

	return dst
}

// CheckSignature checks if the signature is valid for the id (which is a hash
// of the serialized event content). returns an error if the signature itself is
// invalid.
func (ev *Event) CheckSignature() (valid bool, e error) {

	// read and check pubkey
	var pkBytes []byte
	if pkBytes, e = hex.DecodeString(ev.PubKey); fails(e) {
		return false, fmt.Errorf("event pubkey '%s' is invalid hex: %w",
			ev.PubKey, e)
	}

	var pk *secp256k1.PublicKey
	if pk, e = schnorr.ParsePubKey(pkBytes); e != nil {
		return false, fmt.Errorf("event has invalid pubkey '%s': %w",
			ev.PubKey, e)
	}

	// read signature
	var sigBytes []byte
	if sigBytes, e = hex.DecodeString(ev.Sig); e != nil {
		return false, fmt.Errorf("signature '%s' is invalid hex: %w",
			ev.Sig, e)
	}
	var sig *schnorr.Signature
	sig, e = schnorr.ParseSignature(sigBytes)
	if e != nil {
		return false, fmt.Errorf("failed to parse signature: %w", e)
	}

	// check signature
	hash := sha256.Sum256(ev.Serialize())
	return sig.Verify(hash[:], pk), nil
}

// Sign signs an event with a given SecretKey.
func (ev *Event) Sign(skStr string,
	signOpts ...schnorr.SignOption) (e error) {

	var skBytes []byte
	skBytes, e = hex.DecodeString(skStr)
	if e != nil {
		return fmt.Errorf("sign called with invalid secret key '%s': %w",
			skStr, e)
	}

	if ev.Tags == nil {
		ev.Tags = make(tags.T, 0)
	}

	sk, pk := ec.SecKeyFromBytes(skBytes)
	pkBytes := schnorr.SerializePubKey(pk)
	ev.PubKey = hex.EncodeToString(pkBytes[1:])

	h := sha256.Sum256(ev.Serialize())
	var sig *schnorr.Signature
	sig, e = schnorr.Sign(sk, h[:], signOpts...)
	if e != nil {
		return e
	}

	ev.ID = hex.EncodeToString(h[:])
	ev.Sig = hex.EncodeToString(sig.Serialize())

	return nil
}
