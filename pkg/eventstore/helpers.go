package eventstore

import (
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
)

func isOlder(previous, next *event.T) bool {
	return previous.CreatedAt < next.CreatedAt ||
		(previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}
