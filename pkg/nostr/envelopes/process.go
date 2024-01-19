package envelopes

import (
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
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/sentinel"
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

	var match string
	if match, buf, e = sentinel.Identify(b); log.Fail(e) {
		return
	}
	// if env, e = sentinel.Read(buf, typ); log.Fail(e) {
	// 	return
	// }

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
