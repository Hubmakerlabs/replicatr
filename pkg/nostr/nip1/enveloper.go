package nip1

import (
	"fmt"
	"github.com/nostric/replicatr/pkg/wire/array"
	"github.com/nostric/replicatr/pkg/wire/text"
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
	Unmarshal(buf *text.Buffer) (e error)

	array.Arrayer
}

// ProcessEnvelope scans a message and if it finds a correctly formed Envelope it
// unmarshals it and returns it.
//
// If it fails, it also returns the label bytes found and the buffer, which will
// have the cursor at the next byte after the quote delimiter of the Label,
// ready for some other envelope outside of nip-01 to decode.
func ProcessEnvelope(b []byte) (env Enveloper, label []byte, buf *text.Buffer,
	e error) {
	// log.D.F("processing envelope:\n%s", string(b))
	// The bytes must be valid JSON but we can't assume they are free of
	// whitespace... So we will use some tools.
	buf = text.New(b)
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
