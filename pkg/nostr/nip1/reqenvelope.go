package nip1

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

// ReqEnvelope is the wrapper for a query to a relay.
type ReqEnvelope struct {
	SubscriptionID SubscriptionID
	Filters
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *ReqEnvelope) Label() (l Label) { return LReq }

func (E *ReqEnvelope) ToArray() (arr array.T) {
	arr = array.T{REQ, E.SubscriptionID}
	for _, f := range E.Filters {
		arr = append(arr, f.ToObject())
	}
	return
}

func (E *ReqEnvelope) String() (s string) {
	return E.ToArray().String()
}

func (E *ReqEnvelope) Bytes() (s []byte) {
	return E.ToArray().Bytes()
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
	// log.D.F("REQ '%s'", buf.Buf[buf.Pos:])
	// Next, find the comma after the label
	if e = buf.ScanThrough(','); e != nil {
		return
	}
	var which byte
	// ReqEnvelope can have one or no subscription IDs, if it is present we want
	// to collect it before looking for the filters.
	which, e = buf.ScanForOneOf(false, '{', '"')
	if which == '"' {
		// Next character we find will be open quotes for the subscription ID.
		if e = buf.ScanThrough('"'); e != nil {
			return
		}
		var sid []byte
		// read the string
		if sid, e = buf.ReadUntil('"'); fails(e) {
			return fmt.Errorf("unterminated quotes in JSON, probably truncated read")
		}
		log.D.F("Subscription ID: '%s'", sid)
		E.SubscriptionID = SubscriptionID(sid)
	}
	// Next, find the comma (there must be one and at least one object brace
	// after it
	if e = buf.ScanThrough(','); e != nil {
		return
	}
	for {
		// find the opening brace of the event object, usually this is the very
		// next character, we aren't checking for valid whitespace because
		// laziness.
		if e = buf.ScanUntil('{'); e != nil {
			return fmt.Errorf("event not found in event envelope")
		}
		// now we should have an event object next. It has no embedded object so
		// it should end with a close brace. This slice will be wrapped in
		// braces and contain paired brackets, braces and quotes.
		var filterArray []byte
		if filterArray, e = buf.ReadEnclosed(); fails(e) {
			return
		}
		// log.D.F("filter: '%s'", filterArray)
		f := &Filter{}
		if e = json.Unmarshal(filterArray, f); fails(e) {
			return
		}
		E.Filters = append(E.Filters, f)
		// log.D.F("remaining: '%s'", buf.Buf[buf.Pos:])
		which = 0
		if which, e = buf.ScanForOneOf(true, ',', ']'); fails(e) {
			return
		}
		// log.D.F("'%s'", string(which))
		if which == ']' {
			break
		}
	}
	return
}
