package nip1

import (
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

// ClosedEnvelope is a wrapper for a signal to cancel a subscription.
type ClosedEnvelope struct {
	SubscriptionID
	Reason string
}

func NewClosedEnvelope(s SubscriptionID, reason string) (ce *ClosedEnvelope) {
	ce = &ClosedEnvelope{SubscriptionID: s, Reason: reason}
	return
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *ClosedEnvelope) Label() (l Label) { return LClose }

func (E *ClosedEnvelope) ToArray() (a array.T) {
	return array.T{CLOSED, E.SubscriptionID, E.Reason}
}

func (E *ClosedEnvelope) String() (s string) {
	return E.ToArray().String()
}

func (E *ClosedEnvelope) Bytes() (s []byte) {
	return E.ToArray().Bytes()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *ClosedEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.Bytes(), nil
}

// Unmarshal the envelope.
func (E *ClosedEnvelope) Unmarshal(buf *text.Buffer) (e error) {
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
	if sid, e = buf.ReadUntil('"'); fails(e) {
		return fmt.Errorf("unterminated quotes in JSON, probably truncated read")
	}
	E.SubscriptionID = SubscriptionID(sid[:])
	// Next must be a string, which can be empty, but must be at minimum a pair
	// of quotes.
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var reason []byte
	if reason, e = buf.ReadUntil('"'); fails(e) {
		return fmt.Errorf("did not find reason value in close envelope")
	}
	E.Reason = string(text.UnescapeByteString(reason))
	return
}
