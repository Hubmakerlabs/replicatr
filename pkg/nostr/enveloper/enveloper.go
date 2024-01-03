package enveloper

import (
	"fmt"

	close2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/close"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/closed"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
	log2 "mleku.online/git/log"
)

var log = log2.GetLogger()
var fails = log.D.Chk

// The Enveloper interface.
//
// Note that the Unmarshal function is not UnmarshalJSON for a specific reason -
// it is impossible to implement a typed JSON unmarshaler in Go for an array
// type because it must by definition have a sentinel field which in the case of
// nostr is the Label. Objects have a defined collection of recognised labels
// and with omitempty marking the mandatory ones, acting as a "kind" of sentinel.
type Enveloper interface {

	// Label returns the label enum/type of the envelope. The relevant bytes could
	// be retrieved using nip1.List[T]
	Label() labels.T

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
// have the cursor at the next byte after the quote delimiter of the T,
// ready for some other envelope outside of nip-01 to decode.
func ProcessEnvelope(b []byte) (env Enveloper, label []byte, buf *text.Buffer,
	e error) {
	log.D.F("processing envelope:\n%s", string(b))
	// The bytes must be valid JSON but we can't assume they are free of
	// whitespace... So we will use some tools.
	buf = text.NewBuffer(b)
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
	// log.D.F("label: '%s' %v", string(candidate), List)
	var differs bool
	var match labels.T
matched:
	for i := range labels.List {
		differs = false
		if len(candidate) == len(labels.List[i]) {
			for j := range candidate {
				if candidate[j] != labels.List[i][j] {
					differs = true
					break
				}
			}
			if !differs {
				// there can only be one!
				match = i
				break matched
			}
		}
	}
	// if there was no match we still have zero.
	if match == labels.LNil {
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
	case labels.LEvent:
		env = &event.Envelope{}
		e = env.Unmarshal(buf)
	case labels.LOK:
		env = &nip1.OKEnvelope{}
		e = env.Unmarshal(buf)
	case labels.LNotice:
		env = &notice.Envelope{}
		e = env.Unmarshal(buf)
	case labels.LEOSE:
		env = &eose.Envelope{}
		e = env.Unmarshal(buf)
	case labels.LClose:
		env = &close2.Envelope{}
		e = env.Unmarshal(buf)
	case labels.LClosed:
		env = &closed.Envelope{}
		e = env.Unmarshal(buf)
	case labels.LReq:
		env = &nip1.ReqEnvelope{}
		e = env.Unmarshal(buf)
	default:
		// we know it is one of the above but static analysers don't.
	}
	return
}
