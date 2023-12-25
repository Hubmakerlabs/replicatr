package nip1

import (
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

// CloseEnvelope is a wrapper for a signal to cancel a subscription.
type CloseEnvelope struct {
	SubscriptionID
}

func NewCloseEnvelope(s SubscriptionID) (ce *CloseEnvelope) {
	return &CloseEnvelope{SubscriptionID: s}
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *CloseEnvelope) Label() (l Label)   { return LClose }
func (E *CloseEnvelope) String() (s string) { return E.ToArray().String() }
func (E *CloseEnvelope) Bytes() (s []byte)  { return E.ToArray().Bytes() }

func (E *CloseEnvelope) ToArray() (a array.T) {
	return array.T{CLOSE, E.SubscriptionID}
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *CloseEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *CloseEnvelope) Unmarshal(buf *text.Buffer) (e error) {
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
	return
}
