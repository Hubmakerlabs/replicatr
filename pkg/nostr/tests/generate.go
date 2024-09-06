package tests

import (
	"encoding/base64"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
	"lukechampine.com/frand"
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
