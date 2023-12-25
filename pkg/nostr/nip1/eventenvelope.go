package nip1

import (
	"encoding/json"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

// EventEnvelope is the wrapper expected by a relay around an event.
type EventEnvelope struct {

	// The SubscriptionID field is optional, and may at most contain 64 characters,
	// sufficient for encoding a 256 bit hash as hex.
	SubscriptionID SubscriptionID

	// The Event is here a pointer because it should not be copied unnecessarily.
	Event *Event
}

// NewEventEnvelope builds an EventEnvelope from a provided SubscriptionID
// string and pointer to an Event, and returns either the EventEnvelope or an
// error if the Subscription ID is invalid or the Event is nil.
func NewEventEnvelope(si string, ev *Event) (ee *EventEnvelope, e error) {
	var sid SubscriptionID
	if sid, e = NewSubscriptionID(si); fails(e) {
		return
	}
	if ev == nil {
		e = fmt.Errorf("cannot make event envelope with nil event")
		return
	}
	return &EventEnvelope{SubscriptionID: sid, Event: ev}, nil
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *EventEnvelope) Label() (l Label) { return LEvent }

// ToArray converts an EventEnvelope to a form that has a JSON formatted String
// and Bytes function (array.T). To get the encoded form, invoke either of these
// methods on the returned value.
func (E *EventEnvelope) ToArray() (a array.T) {

	// Event envelope has max 3 fields
	a = make(array.T, 0, 3)
	a = append(a, EVENT)
	if E.SubscriptionID.IsValid() {
		a = append(a, E.SubscriptionID)
	}
	a = append(a, E.Event.ToObject())
	return
}

func (E *EventEnvelope) String() (s string) {
	return E.ToArray().String()
}

func (E *EventEnvelope) Bytes() (s []byte) {
	return E.ToArray().Bytes()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *EventEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *EventEnvelope) Unmarshal(buf *text.Buffer) (e error) {
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
	// Next, find the comma after the subscription ID (note we aren't checking
	// that only whitespace intervenes because laziness, usually this is the
	// very next
	// character)
	if e = buf.ScanUntil(','); e != nil {
		return fmt.Errorf("event not found in event envelope")
	}
	// find the opening brace of the event object, usually this is the very next
	// character, we aren't checking for valid whitespace because laziness.
	if e = buf.ScanUntil('{'); e != nil {
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
	E.Event = &Event{}
	if e = json.Unmarshal(eventObj, E.Event); fails(e) {
		log.D.S(string(eventObj))
		return
	}
	// technically we maybe should read ahead further to make sure the JSON
	// closes correctly. Not going to abort because of this.
	if e = buf.ScanUntil(']'); e != nil {
		return fmt.Errorf("malformed JSON, no closing bracket on array")
	}
	// whatever remains doesn't matter as the envelope has fully unmarshaled.
	return
}
