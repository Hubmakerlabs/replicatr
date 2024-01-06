package envelopes

import (
	"fmt"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/OK"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closed"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/req"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

var log, fails = log2.GetStd()

// ProcessEnvelope scans a message and if it finds a correctly formed Envelope
// it unmarshals it and returns it.
//
// If it fails, it also returns the label bytes found and the buffer, which will
// have the cursor at the next byte after the quote delimiter of the T, ready
// for some other envelope outside of nip-01 to decode.
func ProcessEnvelope(b []byte) (env enveloper.Enveloper, label []byte,
	buf *text.Buffer, e error) {

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
		// label is the string that was found in the first element of the JSON
		// array.
		label = candidate
		return
	}
	// We know what to expect now, the next thing to do is pass forward to the
	// specific envelope unmarshaler.
	switch match {
	case labels.EVENT:
		env = &event.Envelope{}
		e = env.Unmarshal(buf)
	case labels.OK:
		env = &OK.Envelope{}
		e = env.Unmarshal(buf)
	case labels.NOTICE:
		env = &notice.Envelope{}
		e = env.Unmarshal(buf)
	case labels.EOSE:
		env = &eose.Envelope{}
		e = env.Unmarshal(buf)
	case labels.CLOSE:
		env = &closer.Envelope{}
		e = env.Unmarshal(buf)
	case labels.CLOSED:
		env = &closed.Envelope{}
		e = env.Unmarshal(buf)
	case labels.REQ:
		env = &req.Envelope{}
		e = env.Unmarshal(buf)
	default:
		// we know it is one of the above but static analysers don't.
	}
	return
}
