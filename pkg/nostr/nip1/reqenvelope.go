package nip1

import (
	"encoding/json"
	"fmt"
	"github.com/nostric/replicatr/pkg/wire/array"
	"github.com/nostric/replicatr/pkg/wire/text"
)

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
func (E *ReqEnvelope) Unmarshal(buf *text.Buffer) (e error) {
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
	if sid, e = buf.ReadUntil('"'); fails(e) {
		return fmt.Errorf("unterminated quotes in JSON, probably truncated read")
	}
	E.SubscriptionID = SubscriptionID(sid)
	// find the opening brace of the event object, usually this is the very next
	// character, we aren't checking for valid whitespace because laziness.
	if e = buf.ScanUntil('['); e != nil {
		return fmt.Errorf("event not found in event envelope")
	}
	// now we should have an event object next. It has no embedded object so it
	// should end with a close brace. This slice will be wrapped in braces and
	// contain paired brackets, braces and quotes.
	var filterArray []byte
	if filterArray, e = buf.ReadThrough(']'); fails(e) {
		return
	}
	log.D.Ln(string(filterArray))
	if e = json.Unmarshal(filterArray, &E.Filters); fails(e) {
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
