package nip1

import (
	"bytes"
	"fmt"
	"io"
	"mleku.online/git/replicatr/pkg/wire/array"
)

type Label = byte

// Label enums for compact identification of the label.
const (
	LabelNil Label = iota
	LabelEvent
	LabelOK
	LabelNotice
	LabelEOSE
	LabelClose
	LabelReq
)

// Labels is the nip1 envelope labels, matching the above enums.
var Labels = [][]byte{
	nil,
	[]byte("EVENT"),
	[]byte("OK"),
	[]byte("NOTICE"),
	[]byte("EOSE"),
	[]byte("CLOSE"),
	[]byte("REQ"),
}

// The Enveloper interface.
//
// Note that the Unmarshal function is not UnmarshalJSON for a specific reason -
// it is impossible to implement a typed JSON unmarshaler in Go for an array
// type because it must by definition have a sentinel field which in the case of
// nostr is the Label. Objects have a defined collection of recognised labels
// and with omitempty marking the mandatory ones, acting as a "kind" of sentinel.
// Javascript is not a serious language and ECMA are not real engineers, and the
// "ninjas" who use javascript are generally ignorant of basic principles of CS.
type Enveloper interface {

	// Label returns the label enum/type of the envelope. The relevant bytes could
	// be retrieved using nip1.Labels[Label]
	Label() Label

	// MarshalJSON returns the JSON encoded form of the envelope.
	MarshalJSON() (bytes []byte, e error)

	// Unmarshal the envelope.
	Unmarshal(buf *bytes.Buffer) (e error)
}

func ReadUntilChar(buf *bytes.Buffer, c byte) (e error) {
	var b byte
	for {
		if b, e = buf.ReadByte(); e == io.EOF {
			return
		}
		if b == c {
			return
		}
	}
}

// ProcessEnvelope scans a message and if it finds a correctly formed Envelope it
// unmarshals it and returns it.
//
// If it fails, it also returns the label bytes found and the buffer, which will
// have the cursor at the next byte after the quote delimiter of the Label,
// ready for some other envelope outside of nip-01 to decode.
func ProcessEnvelope(b []byte) (env Enveloper, label []byte, buf *bytes.Buffer,
	e error) {
	// The bytes must be valid JSON but we can't assume they are free of
	// whitespace... So we will use some tools.
	buf = bytes.NewBuffer(b)
	// First there must be an opening bracket.
	if e = ReadUntilChar(buf, '['); e != nil {
		return
	}
	// Then a quote.
	if e = ReadUntilChar(buf, '"'); e != nil {
		return
	}
	var candidate []byte
	if candidate, e = buf.ReadBytes('"'); e != nil {
		return
	}
	// trim off the quote character.
	candidate = candidate[:len(candidate)-1]
	var differs bool
	var match Label
matched:
	for i := range Labels {
		differs = false
		if len(candidate) == len(Labels[i]) {
			for j := range candidate {
				if candidate[j] != Labels[i][j] {
					differs = true
					break
				}
			}
			if !differs {
				// there can only be one!
				match = Label(i)
				break matched
			}
		}
	}
	// if there was no match we still have zero.
	if match == LabelNil {
		// no match
		e = fmt.Errorf("label '%s' not recognised as nip1 envelope label",
			string(candidate))
		// label is the string that was found in the first element of the JSON
		// array.
		label = candidate
		return
	}
	// We know what to expect now, the next thing to do is pass forward to the specific envelope unmarshaler.
	switch match {
	case LabelEvent:
		env = &EventEnvelope{}
		e = env.Unmarshal(buf)
	case LabelOK:
		env = &OKEnvelope{}
		e = env.Unmarshal(buf)
	case LabelNotice:
		var ne NoticeEnvelope
		env = &ne
		e = env.Unmarshal(buf)
	case LabelEOSE:
		var eose EOSEEnvelope
		env = &eose
		e = env.Unmarshal(buf)
	case LabelClose:
		var c CloseEnvelope
		env = &c
		e = env.Unmarshal(buf)
	case LabelReq:
		env = &ReqEnvelope{}
		e = env.Unmarshal(buf)
	default:
		// we know it is one of the above but static analysers don't.
	}
	return
}

// EventEnvelope is the wrapper expected by a relay around a new event.
type EventEnvelope struct {

	// The SubscriptionID field is optional, and may at most contain 64 characters,
	// sufficient for encoding a 256 bit hash as hex.
	SubscriptionID SubscriptionID

	// The Event is here a pointer because it should not be copied unnecessarily.
	Event Event
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
	return &EventEnvelope{SubscriptionID: sid, Event: *ev}, nil
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (ee *EventEnvelope) Label() (l Label) { return LabelEvent }

// ToArray converts an EventEnvelope to a form that has a JSON formatted String
// and Bytes function (array.T). To get the encoded form, invoke either of these
// methods on the returned value.
func (ee *EventEnvelope) ToArray() (a array.T) {

	// Event envelope has max 3 fields
	a = make(array.T, 0, 3)
	a = append(a, LabelEvent)
	if ee.SubscriptionID.IsValid() {
		a = append(a, ee.SubscriptionID)
	}
	a = append(a, ee.Event.ToObject())
	return
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (ee *EventEnvelope) MarshalJSON() (bytes []byte, e error) {
	return ee.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (ee *EventEnvelope) Unmarshal(buf *bytes.Buffer) (e error) {
	if ee == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}

	return
}

const (
	RejectReasonPOW         = "pow"
	RejectReasonDuplicate   = "duplicate"
	RejectReasonBlocked     = "blocked"
	RejectReasonRateLimited = "rate-limited"
	RejectReasonInvalid     = "invalid"
	RejectReasonError       = "error"
)

// OKEnvelope is a relay message sent in response to an EventEnvelope to
// indicate acceptance (OK is true), rejection and provide a human readable
// Reason for clients to display to users, with the first word being a machine
// readable reason type, as listed in the RejectReason* constants above,
// followed by ": " and a human readable message.
type OKEnvelope struct {
	EventID EventID
	OK      bool
	Reason  string
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (ee *OKEnvelope) Label() (l Label) { return LabelOK }

func NewOKEnvelope(eventID string, ok bool, reason string) (o *OKEnvelope,
	e error) {
	var ei EventID
	if ei, e = NewEventID(eventID); fails(e) {
		return
	}
	o = &OKEnvelope{
		EventID: ei,
		OK:      ok,
		Reason:  reason,
	}
	return
}

// ToArray converts an OKEnvelope to a form that has a JSON formatted String
// and Bytes function (array.T). To get the encoded form, invoke either of these
// methods on the returned value.
func (ee *OKEnvelope) ToArray() (a array.T) {
	return array.T{LabelOK, ee.EventID, ee.OK, ee.Reason}
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (ee *OKEnvelope) MarshalJSON() (bytes []byte, e error) {
	return ee.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (ee *OKEnvelope) Unmarshal(buf *bytes.Buffer) (e error) {
	if ee == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}

	return
}

// ReqEnvelope is the wrapper for a query to a relay.
type ReqEnvelope struct {
	SubscriptionID SubscriptionID
	Filters
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (ee *ReqEnvelope) Label() (l Label) { return LabelReq }

// MarshalJSON returns the JSON encoded form of the envelope.
func (ee *ReqEnvelope) MarshalJSON() (bytes []byte, e error) {
	return ee.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (ee *ReqEnvelope) Unmarshal(buf *bytes.Buffer) (e error) {
	if ee == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}

	return
}

// NoticeEnvelope is a relay message intended to be shown to users in a nostr
// client interface.
type NoticeEnvelope string

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (ee *NoticeEnvelope) Label() (l Label) { return LabelNotice }

func (ne NoticeEnvelope) ToArray() (a array.T) {
	a = array.T{LabelNotice, ne}
	return
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (ee *NoticeEnvelope) MarshalJSON() (bytes []byte, e error) {
	return ee.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (ee *NoticeEnvelope) Unmarshal(buf *bytes.Buffer) (e error) {
	if ee == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}

	return
}

// EOSEEnvelope is a message that indicates that all cached events have been
// delivered and thereafter events will be new and delivered in pubsub subscribe
// fashion while the socket remains open.
type EOSEEnvelope SubscriptionID

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (ee *EOSEEnvelope) Label() (l Label) { return LabelEOSE }

func (ee EOSEEnvelope) ToArray() (a array.T) {
	a = array.T{LabelEOSE, ee}
	return
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (ee *EOSEEnvelope) MarshalJSON() (bytes []byte, e error) {
	return ee.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (ee *EOSEEnvelope) Unmarshal(buf *bytes.Buffer) (e error) {
	if ee == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}

	return
}

// CloseEnvelope is a wrapper for a signal to cancel a subscription.
type CloseEnvelope SubscriptionID

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (ee *CloseEnvelope) Label() (l Label) { return LabelClose }

func (ee CloseEnvelope) ToArray() (a array.T) {
	a = array.T{LabelClose, ee}
	return
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (ee *CloseEnvelope) MarshalJSON() (bytes []byte, e error) {
	return ee.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (ee *CloseEnvelope) Unmarshal(buf *bytes.Buffer) (e error) {
	if ee == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}

	return
}
