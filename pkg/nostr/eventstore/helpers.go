package eventstore

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

func isOlder(prev, next *event.T) bool {
	return prev.CreatedAt < next.CreatedAt ||
		(prev.CreatedAt == next.CreatedAt && prev.ID > next.ID)
}
