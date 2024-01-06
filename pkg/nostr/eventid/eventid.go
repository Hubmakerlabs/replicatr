package eventid

import (
	"encoding/hex"
	"fmt"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

var log, fails = log2.GetStd()

var (
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

// EventID is the SHA256 hash in hexadecimal of the canonical form of an event
// as produced by the output of T.ToCanonical().Bytes().
type EventID string

func (ei EventID) String() string {
	return string(ei)
}

func (ei EventID) Bytes() (b []byte) {
	var e error
	if b, e = hexDecode(string(ei)); log.E.Chk(e) {
		return
	}
	return
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
	// the hash of the canonical representation of the event as per T.ToCanonical())
	if len(ei) != 64 {
		return fmt.Errorf("event ID invalid length: got %d expect 64", len(ei))
	}
	return
}
