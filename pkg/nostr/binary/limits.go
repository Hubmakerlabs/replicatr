package binary

import (
	"math"

	nostr "github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
)

const (
	MaxKind         = math.MaxUint16
	MaxCreatedAt    = math.MaxUint32
	MaxContentSize  = math.MaxUint16
	MaxTagCount     = math.MaxUint16
	MaxTagItemCount = math.MaxUint8
	MaxTagItemSize  = math.MaxUint16
)

func EventEligibleForBinaryEncoding(event *nostr.T) bool {
	if len(event.Content) > MaxContentSize || event.Kind > MaxKind || event.CreatedAt > MaxCreatedAt || len(event.Tags) > MaxTagCount {
		return false
	}

	for _, tag := range event.Tags {
		if len(tag) > MaxTagItemCount {
			return false
		}
		for _, item := range tag {
			if len(item) > MaxTagItemSize {
				return false
			}
		}
	}

	return true
}
