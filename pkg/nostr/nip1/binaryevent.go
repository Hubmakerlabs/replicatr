package nip1

import (
	"mleku.online/git/replicatr/pkg/nostr/tags"
	"mleku.online/git/replicatr/pkg/nostr/time"
)

// BinaryEvent is the primary message type of the nostr protocol, with relevant
// fields encoded as their byte slice form.
//
// The ID hash is generated on the normalised form of a subset of the event
// fields as follows:
//
// [
//
//	0,
//	<pubkey, as a lowercase hex string>,
//	<created_at, as a number>,
//	<kind, as a number>,
//	<tags, as an array of arrays of non-null strings>,
//	<content, as a string>
//
// ]
//
// Note that there is no given justification for the initial zero at the
// beginning of this canonical form, just put it there.
//
// To derive this minimal normalised form, the syntax of JSON is used, and to be
// explicit we repeat here what that means precisely:
//
// - between all of the fields is a comma `,` and no comma after the last
// element
//
// - all spaces are removed
//
// - "number" means base 10 encoded as ASCII
//
// - array means a comma separated list of items between square brackets, with
// no comma after the final item
//
// - string means a string contained between double quotes `"` and containing no
// other double quotes within it. Special characters are handled as per RFC8259
// section 7, and should be processed from raw (ASCII/UTF-8) strings using
// jsontext.EscapeJSONStringAndWrap.
type BinaryEvent struct {

	// ID is 32-bytes lowercase hex-encoded sha256 of the serialized event data
	ID [32]byte `json:"id"`

	// PubKey is a 32-bytes lowercase hex-encoded public key of the event
	// creator
	PubKey [32]byte `json:"pubkey"`

	// CreatedAt is unix timestamp in seconds
	CreatedAt time.Stamp `json:"created_at"`

	// Kind is a 16 bit integer, 0-65535
	Kind uint16 `json:"kind"`

	// Tags are a set of tag identifiers to classify the event
	Tags tags.T `json:"tags"`

	// Content is an arbitrary string containing the body of the event
	Content string `json:"content"`

	// Signature is the BIP 340 Schnorr signature on the ID
	Sig [64]byte `json:"sig"`
}
