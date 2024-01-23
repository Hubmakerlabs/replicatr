package eventid

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

var log = slog.GetStd()

// T is the SHA256 hash in hexadecimal of the canonical form of an event
// as produced by the output of T.ToCanonical().Bytes().
type T string

func (ei T) String() string {
	return string(ei)
}

func (ei T) Bytes() (b []byte) {
	var err error
	if b, err = hex.Dec(string(ei)); log.E.Chk(err) {
		return
	}
	return
}

func (ei T) MarshalJSON() (b []byte, err error) {
	return text.EscapeJSONStringAndWrap(string(ei)), nil
}

// New inspects a string and ensures it is a valid, 64 character long
// hexadecimal string, returns the string coerced to the type.
func New(s string) (ei T, err error) {
	ei = T(s)
	if err = ei.Validate(); log.Fail(err) {

		// clear the result since it failed.
		ei = ei[:0]
		return
	}
	return
}

// Validate checks the T string is valid hex and 64 characters long.
func (ei T) Validate() (err error) {

	// Check the string decodes as valid hexadecimal.
	if _, err = hex.Dec(string(ei)); err != nil {
		return
	}

	// Check the string is 64 bytes long, as an event ID is required to be (it's
	// the hash of the canonical representation of the event as per T.ToCanonical())
	if len(ei) != 64 {
		return fmt.Errorf("event ID invalid length: got %d expect 64", len(ei))
	}
	return
}
