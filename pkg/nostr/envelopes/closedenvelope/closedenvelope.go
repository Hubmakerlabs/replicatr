package closedenvelope

import (
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"mleku.net/slog"
)

var log, chk = slog.New(os.Stderr)

// T is a wrapper for a signal to cancel a subscription.
type T struct {
	ID     subscriptionid.T
	Reason string
}

var _ enveloper.I = &T{}

func New(s subscriptionid.T, reason string) *T {
	return &T{ID: s,
		Reason: reason}
}
func (E *T) ToArray() array.T {
	return array.T{labels.CLOSED,
		E.ID, E.Reason}
}
func (E *T) Label() string                { return labels.CLOSED }
func (E *T) String() (s string)           { return E.ToArray().String() }
func (E *T) Bytes() (s []byte)            { return E.ToArray().Bytes() }
func (E *T) MarshalJSON() ([]byte, error) { return E.Bytes(), nil }

// Unmarshal the envelope.
func (E *T) Unmarshal(buf *text.Buffer) (err error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label (note we aren't checking that only
	// whitespace intervenes because laziness, usually this is the very next
	// character).
	if err = buf.ScanUntil(','); err != nil {
		return
	}
	// Next character we find will be open quotes for the subscription ID.
	if err = buf.ScanThrough('"'); err != nil {
		return
	}
	var sid []byte
	// read the string
	if sid, err = buf.ReadUntil('"'); chk.D(err) {
		return fmt.Errorf("unterminated quotes in JSON, probably truncated read: %s",
			err)
	}
	E.ID = subscriptionid.T(sid[:])
	// Next must be a string, which can be empty, but must be at minimum a pair
	// of quotes.
	if err = buf.ScanThrough('"'); err != nil {
		return
	}
	var reason []byte
	if reason, err = buf.ReadUntil('"'); chk.D(err) {
		return fmt.Errorf("did not find reason value in close envelope: %s",
			err)
	}
	E.Reason = string(text.UnescapeByteString(reason))
	return
}
