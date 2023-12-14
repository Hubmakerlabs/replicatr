package nip1

import (
	"encoding/json"
	"fmt"
	"mleku.online/git/mangle"
	"mleku.online/git/replicatr/pkg/wire/array"
	"reflect"
	"sort"
)

type Label = byte

// Label enums for compact identification of the label.
const (
	LNil Label = iota
	LEvent
	LOK
	LNotice
	LEOSE
	LClose
	LReq
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

// With these, labels have easy short names for the strings, as well as neat
// consistent 1 byte enum version. Having all 3 versions also makes writing the
// recogniser easier.
var (
	EVENT  = string(Labels[LEvent])
	OK     = string(Labels[LOK])
	REQ    = string(Labels[LReq])
	NOTICE = string(Labels[LNotice])
	EOSE   = string(Labels[LEOSE])
	CLOSE  = string(Labels[LClose])
)

func GetLabel(s string) (l Label) {
	for i := range Labels {
		if i == 0 {
			// skip the nil value
			continue
		}
		if string(Labels[i]) == s {
			return Label(i)
		}
	}
	//
	return
}

// The Enveloper interface.
//
// Note that the Unmarshal function is not UnmarshalJSON for a specific reason -
// it is impossible to implement a typed JSON unmarshaler in Go for an array
// type because it must by definition have a sentinel field which in the case of
// nostr is the Label. Objects have a defined collection of recognised labels
// and with omitempty marking the mandatory ones, acting as a "kind" of sentinel.
type Enveloper interface {

	// Label returns the label enum/type of the envelope. The relevant bytes could
	// be retrieved using nip1.Labels[Label]
	Label() Label

	// MarshalJSON returns the JSON encoded form of the envelope.
	MarshalJSON() (bytes []byte, e error)

	// Unmarshal the envelope.
	Unmarshal(buf *mangle.Buffer) (e error)

	array.Arrayer
}

// ProcessEnvelope scans a message and if it finds a correctly formed Envelope it
// unmarshals it and returns it.
//
// If it fails, it also returns the label bytes found and the buffer, which will
// have the cursor at the next byte after the quote delimiter of the Label,
// ready for some other envelope outside of nip-01 to decode.
func ProcessEnvelope(b []byte) (env Enveloper, label []byte, buf *mangle.Buffer,
	e error) {
	// The bytes must be valid JSON but we can't assume they are free of
	// whitespace... So we will use some tools.
	buf = mangle.New(b)
	// First there must be an opening bracket.
	if e = buf.ScanThrough('['); e != nil {
		return
	}
	// Then a quote.
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var candidate []byte
	if candidate, e = buf.ReadUntil('"'); e != nil {
		return
	}
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
	if match == LNil {
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
	case LEvent:
		env = &EventEnvelope{}
		e = env.Unmarshal(buf)
	case LOK:
		env = &OKEnvelope{}
		e = env.Unmarshal(buf)
	case LNotice:
		env = &NoticeEnvelope{}
		e = env.Unmarshal(buf)
	case LEOSE:
		env = &EOSEEnvelope{}
		e = env.Unmarshal(buf)
	case LClose:
		env = &CloseEnvelope{}
		e = env.Unmarshal(buf)
	case LReq:
		env = &ReqEnvelope{}
		e = env.Unmarshal(buf)
	default:
		// we know it is one of the above but static analysers don't.
	}
	return
}

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

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *EventEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *EventEnvelope) Unmarshal(buf *mangle.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label
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
	// Next, find the comma after the subscription ID
	if e = buf.ScanUntil(','); e != nil {
		return fmt.Errorf("event not found in event envelope")
	}
	// find the opening brace of the event object.
	if e = buf.ScanUntil('{'); e != nil {
		return fmt.Errorf("event not found in event envelope")
	}
	// now we should have an event object next. It has no embedded object so it
	// should end with a close brace.
	var eventObj []byte
	if eventObj, e = buf.ReadThrough('}'); fails(e) {
		return
	}
	// TODO: Handle } inside the Content field. Tags also maybe.

	// allocate an event to unmarshal into
	E.Event = &Event{}
	if e = json.Unmarshal(eventObj, E.Event); fails(e) {
		log.D.S(string(eventObj))
		return
	}
	// technically we maybe should read ahead further to make sure the JSON
	// closes correctly.
	if e = buf.ScanUntil(']'); e != nil {
		return fmt.Errorf("malformed JSON, no closing bracket on array")
	}
	// whatever remains doesn't matter as the envelope has fully unmarshaled.
	return
}

const (
	OKPoW         = "pow"
	OKDuplicate   = "duplicate"
	OKBlocked     = "blocked"
	OKRateLimited = "rate-limited"
	OKInvalid     = "invalid"
	OKError       = "error"
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

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *OKEnvelope) Label() (l Label) { return LOK }

// ToArray converts an OKEnvelope to a form that has a JSON formatted String
// and Bytes function (array.T). To get the encoded form, invoke either of these
// methods on the returned value.
func (E *OKEnvelope) ToArray() (a array.T) {
	return array.T{OK, E.EventID, E.OK, E.Reason}
}

func (E *OKEnvelope) String() (s string) {
	return E.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *OKEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

const (
	Btrue     = "true"
	BtrueLen  = len(Btrue)
	Bfalse    = "false"
	BfalseLen = len(Bfalse)
)

// Unmarshal the envelope.
func (E *OKEnvelope) Unmarshal(buf *mangle.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label
	if e = buf.ScanThrough(','); e != nil {
		return
	}
	// next comes an event ID
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var eventID []byte
	if eventID, e = buf.ReadUntil('"'); fails(e) {
		return fmt.Errorf("did not find event ID value in ok envelope")
	}
	// check event is a valid length
	if len(eventID) != 64 {
		return fmt.Errorf("event ID in ok envelope invalid length: %d '%s'",
			len(eventID)-1, string(eventID))
	}
	eventID = eventID[:len(eventID)]
	// check that it's actually hexadecimal
	const hexChars = "0123456789abcdefABCDEF"
	tmp := make([]byte, 64)
	copy(tmp, eventID)
	// this sort is backwards because invalid characters are more likely after
	// the set of hex numbers than before, and the error will be found sooner
	// and shorten the iteration below.
	sort.Slice(tmp, func(i, j int) bool { return tmp[i] > tmp[j] })
next:
	for j := range tmp {
		inSet := false
		for i := range hexChars {
			if hexChars[i] == tmp[j] {
				inSet = true
				continue next
			}
		}
		// if a character in tmp didn't match by the end of hexChars we found an invalid character.
		if !inSet {
			return fmt.Errorf("found non-hex character in event ID: '%s'",
				string(eventID))
		}
	}
	E.EventID = EventID(eventID[:len(eventID)-1])
	// next another comma
	if e = buf.ScanThrough(','); e != nil {
		return
	}
	// next comes a boolean value
	var isOK []byte
	if isOK, e = buf.ReadUntil(','); fails(e) {
		return fmt.Errorf("did not find OK value in ok envelope")
	}
	isOK = isOK[:len(isOK)-1]
	// determine the value encoded
	l := len(isOK)
	var isBool bool
maybeOK:
	switch {
	case l == BtrueLen:
		for i := range isOK {
			if isOK[i] != Btrue[i] {
				break maybeOK
			}
		}
		E.OK = true
		isBool = true
	case l == BfalseLen:
		for i := range isOK {
			if isOK[i] != Bfalse[i] {
				break maybeOK
			}
		}
		isBool = true
	}
	if !isBool {
		return fmt.Errorf("unexpected string in ok envelope OK field '%s'",
			string(isOK))
	}
	// Next must be a string, which can be empty, but must be at minimum a pair
	// of quotes.
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var reason []byte
	if reason, e = buf.ReadUntil('"'); fails(e) {
		return fmt.Errorf("did not find reason value in ok envelope")
	}
	// Scan for the proper envelope ending.
	if e = buf.ScanThrough(']'); e != nil {
		log.D.Ln("envelope unterminated but all fields found")
	}
	E.Reason = string(reason[:len(reason)-1])
	log.D.Ln(E, "\n", buf.String())
	return
}

// ReqEnvelope is the wrapper for a query to a relay.
type ReqEnvelope struct {
	SubscriptionID SubscriptionID
	Filters
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *ReqEnvelope) Label() (l Label) { return LReq }

func (E *ReqEnvelope) ToArray() array.T {
	return array.T{REQ, E.SubscriptionID, E.Filters}
}
func (E *ReqEnvelope) String() (s string) {
	return E.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *ReqEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *ReqEnvelope) Unmarshal(buf *mangle.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label
	if e = buf.ScanThrough(','); e != nil {
		return
	}
	// Next character we find will be open quotes for the subscription ID.
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var sid []byte
	// read the string
	if sid, e = buf.ReadBytes('"'); fails(e) {
		return fmt.Errorf("unterminated quotes in JSON, probably truncated read")
	}
	E.SubscriptionID = SubscriptionID(sid[:len(sid)-1])

	return
}

// NoticeEnvelope is a relay message intended to be shown to users in a nostr
// client interface.
type NoticeEnvelope struct {
	Text string
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *NoticeEnvelope) Label() (l Label) { return LNotice }

func (E *NoticeEnvelope) ToArray() (a array.T) {
	return array.T{NOTICE,
		E.Text}
}
func (E *NoticeEnvelope) String() (s string) {
	return E.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *NoticeEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *NoticeEnvelope) Unmarshal(buf *mangle.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	log.I.Ln(reflect.TypeOf(E))
	return
}

// EOSEEnvelope is a message that indicates that all cached events have been
// delivered and thereafter events will be new and delivered in pubsub subscribe
// fashion while the socket remains open.
type EOSEEnvelope struct {
	SubscriptionID
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *EOSEEnvelope) Label() (l Label) { return LEOSE }

func (E *EOSEEnvelope) ToArray() (a array.T) {
	a = array.T{EOSE, E.SubscriptionID}
	return
}

func (E *EOSEEnvelope) String() (s string) {
	return E.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *EOSEEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *EOSEEnvelope) Unmarshal(buf *mangle.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	log.I.Ln(reflect.TypeOf(E))
	return
}

// CloseEnvelope is a wrapper for a signal to cancel a subscription.
type CloseEnvelope struct {
	SubscriptionID
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *CloseEnvelope) Label() (l Label) { return LClose }

func (E *CloseEnvelope) ToArray() (a array.T) {
	return array.T{CLOSE, E.SubscriptionID}
}

func (E *CloseEnvelope) String() (s string) {
	return E.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *CloseEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *CloseEnvelope) Unmarshal(buf *mangle.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	log.I.Ln(reflect.TypeOf(E))
	return
}
