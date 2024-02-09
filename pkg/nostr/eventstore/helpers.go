package eventstore

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"mleku.online/git/slog"
)

var log, chk = slog.New(os.Stderr)

func isOlder(previous, next *event.T) bool {
	return previous.CreatedAt < next.CreatedAt ||
		(previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}
