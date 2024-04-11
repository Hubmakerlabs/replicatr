package eventest

import (
	"encoding/base64"
	"os"

	"lukechampine.com/frand"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/nostrbinary"
	"mleku.dev/git/nostr/timestamp"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

func GenerateEvent(nsec string, maxSize int) (ev *event.T, binSize int,
	err error) {
	l := frand.Intn(maxSize * 6 / 8) // account for base64 expansion
	ev = &event.T{
		Kind:      kind.TextNote,
		CreatedAt: timestamp.Now(),
		Content:   base64.StdEncoding.EncodeToString(frand.Bytes(l)),
	}
	if err = ev.Sign(nsec); chk.E(err) {
		return
	}
	var bin []byte
	bin, err = nostrbinary.Marshal(ev)
	binSize = len(bin)
	return
}
