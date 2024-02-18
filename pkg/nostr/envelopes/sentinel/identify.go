package sentinel

import (
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"mleku.dev/git/slog"
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
	if err = buf.ScanThrough('['); log.D.Chk(err) {
		return
	}
	// Then a quote.
	if err = buf.ScanThrough('"'); log.D.Chk(err) {
		return
	}
	var candidate []byte
	if candidate, err = buf.ReadUntil('"'); log.D.Chk(err) {
		return
	}
	// log.D.F("label: '%s' %v", string(candidate), List)
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
		err = fmt.Errorf("label '%s' not recognised as envelope label",
			string(candidate))
		return
	}
	trunc := make([]byte, 1024)
	copy(trunc, buf.Buf)
	// log.D.F("received %s envelope '%s'", match, trunc)
	return
}
