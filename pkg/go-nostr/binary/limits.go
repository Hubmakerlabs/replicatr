package binary

import (
	"math"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
)

const (
	MaxKind         = math.MaxUint16
	MaxCreatedAt    = math.MaxUint32
	MaxContentSize  = math.MaxUint16
	MaxTagCount     = math.MaxUint16
	MaxTagItemCount = math.MaxUint8
	MaxTagItemSize  = math.MaxUint16
)

func EventEligibleForBinaryEncoding(evt *event.T) bool {
	if len(evt.Content) > MaxContentSize || evt.Kind > MaxKind || evt.CreatedAt > MaxCreatedAt || len(evt.Tags) > MaxTagCount {
		return false
	}

	for _, tag := range evt.Tags {
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
