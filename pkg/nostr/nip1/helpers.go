package nip1

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/minio/sha256-simd"
	"github.com/mleku/replicatr/pkg/wire/text"
	log2 "mleku.online/git/log"
)

var (
	log                    = log2.GetLogger()
	fails                  = log.D.Chk
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

func Hash(in []byte) (out []byte) {
	h := sha256.Sum256(in)
	return h[:]
}

// SubscriptionID is an arbitrary string of 1-64 characters in length generated
// as a request or session identifier.
type SubscriptionID string

func (si SubscriptionID) MarshalJSON() (b []byte, e error) {
	return text.EscapeJSONStringAndWrap(string(si)), nil
}

// NewSubscriptionID inspects a string and converts to SubscriptionID if it is
// valid. Invalid means length < 0 and <= 64 (hex encoded 256 bit hash).
func NewSubscriptionID(s string) (SubscriptionID, error) {
	si := SubscriptionID(s)
	if si.IsValid() {
		return si, nil
	} else {
		// remove invalid return value
		return si[:0],
			errors.New("invalid subscription ID - either < 0 or > 64 char length")
	}
}

// IsValid returns true if the subscription id is between 1 and 64 characters.
// Invalid means too long or not present.
func (si SubscriptionID) IsValid() bool { return len(si) <= 64 }

// EventID is the SHA256 hash in hexadecimal of the canonical form of an event
// as produced by the output of Event.ToCanonical().Bytes().
type EventID string

func (ei EventID) String() string {
	return string(ei)
}

func (ei EventID) MarshalJSON() (b []byte, e error) {
	return text.EscapeJSONStringAndWrap(string(ei)), nil
}

// NewEventID inspects a string and ensures it is a valid, 64 character long
// hexadecimal string, returns the string coerced to the type.
func NewEventID(s string) (ei EventID, e error) {
	ei = EventID(s)
	if e = ei.Validate(); fails(e) {

		// clear the result since it failed.
		ei = ei[:0]
		return
	}
	return
}

// Validate checks the EventID string is valid hex and 64 characters long.
func (ei EventID) Validate() (e error) {

	// Check the string decodes as valid hexadecimal.
	if _, e = hexDecode(string(ei)); e != nil {
		return
	}

	// Check the string is 64 bytes long, as an event ID is required to be (it's
	// the hash of the canonical representation of the event as per Event.ToCanonical())
	if len(ei) != 64 {
		return fmt.Errorf("event ID invalid length: got %d expect 64", len(ei))
	}
	return
}
