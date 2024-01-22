package envelopes

import (
	"github.com/Hubmakerlabs/replicatr/pkg/interfaces/enveloper"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
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

	log.T.F("processing envelope:\n%s", string(b))
	var match string
	if match, buf, e = sentinel.Identify(b); log.Fail(e) {
		return
	}
	if env, e = sentinel.Read(buf, match); log.Fail(e) {
		return
	}
	return
}
