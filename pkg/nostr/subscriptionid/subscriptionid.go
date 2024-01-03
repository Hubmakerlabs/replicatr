package subscriptionid

import (
	"errors"

	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

// T is an arbitrary string of 1-64 characters in length generated
// as a request or session identifier.
type T string

func (si T) MarshalJSON() (b []byte, e error) {
	return text.EscapeJSONStringAndWrap(string(si)), nil
}

// NewSubscriptionID inspects a string and converts to T if it is
// valid. Invalid means length < 0 and <= 64 (hex encoded 256 bit hash).
func NewSubscriptionID(s string) (T, error) {
	si := T(s)
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
func (si T) IsValid() bool { return len(si) <= 64 && len(si) > 0 }
