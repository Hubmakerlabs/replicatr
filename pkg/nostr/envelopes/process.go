package envelopes

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/sentinel"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

var log, chk = slog.New(os.Stderr)

// ProcessEnvelope scans a message and if it finds a correctly formed
// enveloper.I it unmarshals it and returns it.
//
// If it fails, it also returns the label bytes found and the buffer, which will
// have the cursor at the next byte after the quote delimiter of the T, ready
// for some other envelope outside of nip-01 to decode.
func ProcessEnvelope(b []byte) (env enveloper.I, buf *text.Buffer, err error) {
	var match string
	if match, buf, err = sentinel.Identify(b); log.T.Chk(err) {
		return
	}
	if match == "" {
		err = log.E.Err("cannot read envelope without a label match")
		return
	}
	if env, err = sentinel.Read(buf, match); log.T.Chk(err) {
		return
	}
	return
}
