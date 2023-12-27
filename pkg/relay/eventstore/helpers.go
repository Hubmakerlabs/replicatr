package eventstore

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
)

func isOlder(previous, next *nip1.Event) bool {
	return previous.CreatedAt < next.CreatedAt ||
		(previous.CreatedAt == next.CreatedAt && previous.ID > next.ID)
}
