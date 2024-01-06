package closed

import (
	"fmt"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

var log, fails = log2.GetStd()

// Envelope is a wrapper for a signal to cancel a subscription.
type Envelope struct {
	subscriptionid.T
	Reason string
}

var _ enveloper.Enveloper = &Envelope{}

func NewClosedEnvelope(s subscriptionid.T, reason string) (ce *Envelope) {
	ce = &Envelope{T: s, Reason: reason}
	return
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *Envelope) Label() (l string) { return labels.CLOSED }

func (E *Envelope) ToArray() (a array.T) {
	return array.T{labels.CLOSED, E.T, E.Reason}
}

func (E *Envelope) String() (s string) {
	return E.ToArray().String()
}

func (E *Envelope) Bytes() (s []byte) {
	return E.ToArray().Bytes()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *Envelope) MarshalJSON() (bytes []byte, e error) {
	return E.Bytes(), nil
}

func (E *Envelope) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

// Unmarshal the envelope.
func (E *Envelope) Unmarshal(buf *text.Buffer) (e error) {
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
	E.T = subscriptionid.T(sid[:])
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
