package closer

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
}

var _ enveloper.Enveloper = &Envelope{}

func NewCloseEnvelope(s subscriptionid.T) (ce *Envelope) {
	return &Envelope{T: s}
}

func (E *Envelope) Label() (l string)  { return labels.CLOSE }
func (E *Envelope) String() (s string) { return E.ToArray().String() }
func (E *Envelope) Bytes() (s []byte)  { return E.ToArray().Bytes() }

func (E *Envelope) ToArray() (a array.T) {
	return array.T{labels.CLOSE, E.T}
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *Envelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
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
	return
}
