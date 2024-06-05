package sentinel

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"mleku.net/slog"
)

var log, chk = slog.New(os.Stderr)

// Identify takes a byte slice and scans it as a nostr Envelope array, and
// returns the label type and a text.Buffer that is ready for the Read function
// to generate the appropriate structure.
func Identify(b []byte) (match string, buf *text.Buffer, err error) {
	// The bytes must be valid JSON but we can't assume they are free of
	// whitespace... So we will use some tools.
	buf = text.NewBuffer(b)
	// First there must be an opening bracket.
	if err = buf.ScanThrough('['); chk.T(err) {
		return
	}
	// Then a quote.
	if err = buf.ScanThrough('"'); chk.T(err) {
		return
	}
	var candidate []byte
	if candidate, err = buf.ReadUntil('"'); chk.T(err) {
		return
	}
	if len(candidate) == 0 {
		err = log.E.Err("cannot read envelope without a label\n%s", string(b))
		return
	}
	var differs bool
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
		err = log.E.Err("label '%s' not recognised as envelope label\n%s",
			string(candidate), buf.String())
		return
	}
	return
}
