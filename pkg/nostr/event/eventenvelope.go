package event

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

// Envelope is the wrapper expected by a relay around an event.
type Envelope struct {

	// The SubscriptionID field is optional, and may at most contain 64 characters,
	// sufficient for encoding a 256 bit hash as hex.
	SubscriptionID subscriptionid.T

	// The Event is here a pointer because it should not be copied unnecessarily.
	Event *T
}

// NewEventEnvelope builds an Envelope from a provided T
// string and pointer to an T, and returns either the Envelope or an
// error if the Subscription ID is invalid or the T is nil.
func NewEventEnvelope(si string, ev *T) (ee *Envelope, e error) {
	var sid subscriptionid.T
	if sid, e = subscriptionid.NewSubscriptionID(si); fails(e) {
		return
	}
	if ev == nil {
		e = fmt.Errorf("cannot make event envelope with nil event")
		return
	}
	return &Envelope{SubscriptionID: sid, Event: ev}, nil
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.List[T]
func (E *Envelope) Label() (l labels.T) { return labels.LEvent }

// ToArray converts an Envelope to a form that has a JSON formatted String
// and Bytes function (array.T). To get the encoded form, invoke either of these
// methods on the returned value.
func (E *Envelope) ToArray() (a array.T) {

	// T envelope has max 3 fields
	a = make(array.T, 0, 3)
	a = append(a, labels.EVENT)
	if E.SubscriptionID.IsValid() {
		a = append(a, E.SubscriptionID)
	}
	a = append(a, E.Event.ToObject())
	return
}

func (E *Envelope) String() (s string) {
	return E.ToArray().String()
}

func (E *Envelope) Bytes() (s []byte) {
	return E.ToArray().Bytes()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *Envelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *Envelope) Unmarshal(buf *text.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label (note we aren't checking that only
	// whitespace intervenes because laziness, usually this is the very next
	// character).
	if e = buf.ScanUntil(','); log.D.Chk(e) {
		return
	}
	// Next character we find will be open quotes for the subscription ID, or
	// the open brace of the embedded event.
	var matched byte
	if matched, e = buf.ScanForOneOf(false, '"', '{'); log.D.Chk(e) {
		return
	}
	if matched == '"' {
		// Advance the cursor to consume the quote character.
		buf.Pos++
		var sid []byte
		// Read the string.
		if sid, e = buf.ReadUntil('"'); log.D.Chk(e) {
			return fmt.Errorf("unterminated quotes in JSON, probably truncated read")
		}
		E.SubscriptionID = subscriptionid.T(sid[:])
		// Next, find the comma after the subscription ID (note we aren't checking
		// that only whitespace intervenes because laziness, usually this is the
		// very next character).
		if e = buf.ScanUntil(','); log.D.Chk(e) {
			return fmt.Errorf("event not found in event envelope")
		}
	}
	// find the opening brace of the event object, usually this is the very next
	// character, we aren't checking for valid whitespace because laziness.
	if e = buf.ScanUntil('{'); log.D.Chk(e) {
		return fmt.Errorf("event not found in event envelope")
	}
	// now we should have an event object next. It has no embedded object so it
	// should end with a close brace. This slice will be wrapped in braces and
	// contain paired brackets, braces and quotes.
	var eventObj []byte
	if eventObj, e = buf.ReadEnclosed(); fails(e) {
		return
	}
	// allocate an event to unmarshal into
	E.Event = &T{}
	if e = json.Unmarshal(eventObj, E.Event); fails(e) {
		log.D.S(string(eventObj))
		return
	}
	// technically we maybe should read ahead further to make sure the JSON
	// closes correctly. Not going to abort because of this.
	if e = buf.ScanUntil(']'); log.D.Chk(e) {
		return fmt.Errorf("malformed JSON, no closing bracket on array")
	}
	// whatever remains doesn't matter as the envelope has fully unmarshaled.
	return
}
