package eventstore

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"mleku.online/git/slog"
)

var log = slog.GetStd()

func isOlder(previous, next *event.T) bool {
	return previous.CreatedAt < next.CreatedAt ||
		(previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}
