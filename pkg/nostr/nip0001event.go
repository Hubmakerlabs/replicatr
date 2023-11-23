package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/mleku/replicatr/pkg/jsontext"
	"github.com/mleku/replicatr/pkg/nostr/kind"

	"github.com/mailru/easyjson"
	btcec "github.com/mleku/ec"
	"github.com/mleku/ec/schnorr"
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
	CreatedAt Timestamp `json:"created_at"`

	// Kind is the nostr protocol code for the type of event. See kind.T
	Kind kind.T `json:"kind"`

	// Tags are a list of tags, which are a list of strings usually structured
	// as a 3 layer scheme indicating specific features of an event.
	Tags Tags `json:"tags"`

	// Content is an arbitrary string that can contain anything, but usually
	// conforming to a specification relating to the Kind and the Tags.
	Content string `json:"content"`

	// Sig is the signature on the ID hash that validates as coming from the
	// Pubkey.
	Sig string `json:"sig"`

	// anything here will be mashed together with the main event object when
	// serializing (todo WHO THE FUCK WROTE THIS COMMENT?)
	extra map[string]any
}

// Event Stringer interface, just returns the raw JSON as a string.
func (evt *Event) String() string {
	j, _ := easyjson.Marshal(evt)
	return string(j)
}

// GetID serializes and returns the event ID as a string.
func (evt *Event) GetID() string {
	h := sha256.Sum256(evt.Serialize())
	return hex.EncodeToString(h[:])
}

// Serialize outputs a byte array that can be hashed/signed to
// identify/authenticate. JSON encoding as defined in RFC4627.
func (evt *Event) Serialize() []byte {

	// the serialization process is just putting everything into a JSON array
	// so the order is kept. See NIP-01
	dst := make([]byte, 0)

	// the header portion is easy to serialize
	// [0,"pubkey",created_at,kind,[
	dst = append(dst, []byte(
		fmt.Sprintf(
			"[0,\"%s\",%d,%d,",
			evt.PubKey,
			evt.CreatedAt,
			evt.Kind,
		))...)

	// tags
	dst = evt.Tags.MarshalTo(dst)
	dst = append(dst, ',')

	// content needs to be escaped in general as it is user generated.
	dst = append(dst, jsontext.EscapeJSONStringAndWrap(evt.Content)...)
	dst = append(dst, ']')

	return dst
}

// CheckSignature checks if the signature is valid for the id (which is a hash
// of the serialized event content). returns an error if the signature itself is
// invalid.
func (evt *Event) CheckSignature() (bool, error) {

	// read and check pubkey
	pk, err := hex.DecodeString(evt.PubKey)
	if err != nil {
		return false, fmt.Errorf("event pubkey '%s' is invalid hex: %w",
			evt.PubKey, err)
	}

	pubkey, err := schnorr.ParsePubKey(pk)
	if err != nil {
		return false, fmt.Errorf("event has invalid pubkey '%s': %w",
			evt.PubKey, err)
	}

	// read signature
	s, err := hex.DecodeString(evt.Sig)
	if err != nil {
		return false, fmt.Errorf("signature '%s' is invalid hex: %w", evt.Sig,
			err)
	}
	sig, err := schnorr.ParseSignature(s)
	if err != nil {
		return false, fmt.Errorf("failed to parse signature: %w", err)
	}

	// check signature
	hash := sha256.Sum256(evt.Serialize())
	return sig.Verify(hash[:], pubkey), nil
}

// Sign signs an event with a given SecretKey.
func (evt *Event) Sign(secretKey string,
	signOpts ...schnorr.SignOption) error {
	s, err := hex.DecodeString(secretKey)
	if err != nil {
		return fmt.Errorf("sign called with invalid secret key '%s': %w",
			secretKey, err)
	}

	if evt.Tags == nil {
		evt.Tags = make(Tags, 0)
	}

	sk, pk := btcec.SecKeyFromBytes(s)
	pkBytes := pk.SerializeCompressed()
	evt.PubKey = hex.EncodeToString(pkBytes[1:])

	h := sha256.Sum256(evt.Serialize())
	sig, err := schnorr.Sign(sk, h[:], signOpts...)
	if err != nil {
		return err
	}

	evt.ID = hex.EncodeToString(h[:])
	evt.Sig = hex.EncodeToString(sig.Serialize())

	return nil
}
