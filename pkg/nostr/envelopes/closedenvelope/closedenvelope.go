package closedenvelope

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/interfaces/enveloper"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	l "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

var log = log2.GetStd()

// T is a wrapper for a signal to cancel a subscription.
type T struct {
	ID     subscriptionid.T
	Reason string
}

var _ enveloper.I = &T{}

func (E *T) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

func New(s subscriptionid.T, reason string) *T {
	return &T{ID: s, Reason: reason}
}

func (E *T) ToArray() array.T { return array.T{l.CLOSED, E.ID, E.Reason} }

func (E *T) Label() string { return l.CLOSED }

func (E *T) String() (s string) { return E.ToArray().String() }

func (E *T) Bytes() (s []byte) { return E.ToArray().Bytes() }

func (E *T) MarshalJSON() ([]byte, error) { return E.Bytes(), nil }

// Unmarshal the envelope.
func (E *T) Unmarshal(buf *text.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label (note we aren't checking that only
	// whitespace intervenes because laziness, usually this is the very next
	// character).
	if e = buf.ScanUntil(','); e != nil {
		return
	}
	// Next character we find will be open quotes for the subscription ID.
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var sid []byte
	// read the string
	if sid, e = buf.ReadUntil('"'); log.Fail(e) {
		return fmt.Errorf("unterminated quotes in JSON, probably truncated read")
	}
	E.ID = subscriptionid.T(sid[:])
	// Next must be a string, which can be empty, but must be at minimum a pair
	// of quotes.
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var reason []byte
	if reason, e = buf.ReadUntil('"'); log.Fail(e) {
		return fmt.Errorf("did not find reason value in close envelope")
	}
	E.Reason = string(text.UnescapeByteString(reason))
	return
}
