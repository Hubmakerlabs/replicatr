package nip1

import (
	"fmt"
	"github.com/nostric/replicatr/pkg/wire/array"
	"github.com/nostric/replicatr/pkg/wire/text"
	"sort"
)

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
	// log.D.F("'%s' %v '%s' ", E.EventID, E.OK, E.Reason)
	return array.T{OK, E.EventID, E.OK, E.Reason}
}

func (E *OKEnvelope) String() (s string) {
	return E.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *OKEnvelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("ok envelope marshal")
	return E.ToArray().Bytes(), nil
}

const (
	Btrue     = "true"
	BtrueLen  = len(Btrue)
	Bfalse    = "false"
	BfalseLen = len(Bfalse)
)

// Unmarshal the envelope.
func (E *OKEnvelope) Unmarshal(buf *text.Buffer) (e error) {
	log.D.Ln("ok envelope unmarshal", string(buf.Buf))
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
	E.EventID = EventID(eventID)
	// next another comma
	if e = buf.ScanThrough(','); e != nil {
		return
	}
	// next comes a boolean value
	var isOK []byte
	if isOK, e = buf.ReadUntil(','); fails(e) {
		return fmt.Errorf("did not find OK value in ok envelope")
	}
	isOK = isOK[:len(isOK)]
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
	E.Reason = string(text.UnescapeByteString(reason))
	return
}
