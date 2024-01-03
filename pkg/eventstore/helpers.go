package eventstore

import (
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr"
)

func isOlder(previous, next *nostr.Event) bool {
	return previous.CreatedAt < next.CreatedAt ||
		(previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}
