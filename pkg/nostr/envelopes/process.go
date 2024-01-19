package envelopes

import (
	"fmt"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closedenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eoseenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/noticeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

var log = log2.GetStd()

// ProcessEnvelope scans a message and if it finds a correctly formed
// enveloper.I it unmarshals it and returns it.
//
// If it fails, it also returns the label bytes found and the buffer, which will
// have the cursor at the next byte after the quote delimiter of the T, ready
// for some other envelope outside of nip-01 to decode.
func ProcessEnvelope(b []byte) (env enveloper.I, buf *text.Buffer, e error) {

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
	var match string
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
				match = string(labels.List[i])
				break matched
			}
		}
	}
	// if there was no match we still have zero.
	if match == "" {
		// no match
		e = fmt.Errorf("label '%s' not recognised as envelope label",
			string(candidate))
		return
	}
	// We know what to expect now, the next thing to do is pass forward to the
	// specific envelope unmarshaler.
	switch match {
	case labels.EVENT:
		env = &eventenvelope.T{}
		e = env.Unmarshal(buf)
	case labels.OK:
		env = &okenvelope.T{}
		e = env.Unmarshal(buf)
	case labels.NOTICE:
		env = &noticeenvelope.T{}
		e = env.Unmarshal(buf)
	case labels.EOSE:
		env = &eoseenvelope.T{}
		e = env.Unmarshal(buf)
	case labels.CLOSE:
		env = &closeenvelope.T{}
		e = env.Unmarshal(buf)
	case labels.CLOSED:
		env = &closedenvelope.T{}
		e = env.Unmarshal(buf)
	case labels.REQ:
		env = &reqenvelope.T{}
		e = env.Unmarshal(buf)
	case labels.AUTH:
		log.D.Ln("todo: distinguishing auth envelopes")
	case labels.COUNT:
		log.D.Ln("todo: distinguishing count envelopes")
	default:
		// we know it is one of the above but static analysers don't.
	}
	return
}
