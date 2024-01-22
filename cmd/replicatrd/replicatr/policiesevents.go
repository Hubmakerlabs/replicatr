package replicatr

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"golang.org/x/exp/slices"
)

// PreventTooManyIndexableTags returns a function that can be used as a
// RejectFilter that will reject events with more indexable (single-character)
// tags than the specified number.
//
// If ignoreKinds is given this restriction will not apply to these kinds
// (useful for allowing a bigger). If onlyKinds is given then all other kinds
// will be ignored.
func PreventTooManyIndexableTags(max int, ignoreKinds kinds.T,
	onlyKinds kinds.T) func(context.T, *event.T) (bool, string) {

	ignore := func(kind kind.T) bool { return false }
	if len(ignoreKinds) > 0 {
		ignore = func(k kind.T) bool {
			_, isIgnored := slices.BinarySearch(ignoreKinds, k)
			return isIgnored
		}
	}
	if len(onlyKinds) > 0 {
		ignore = func(k kind.T) bool {
			_, isApplicable := slices.BinarySearch(onlyKinds, k)
			return !isApplicable
		}
	}
	return func(c context.T, event *event.T) (reject bool, msg string) {
		if ignore(event.Kind) {
			return false, ""
		}
		ntags := 0
		for _, tag := range event.Tags {
			if len(tag) > 0 && len(tag[0]) == 1 {
				ntags++
			}
		}
		if ntags > max {
			return true, "too many indexable tags"
		}
		return false, ""
	}
}

// PreventLargeTags rejects events that have indexable tag values greater than
// maxTagValueLen.
func PreventLargeTags(maxTagValueLen int) func(context.T, *event.T) (bool, string) {
	return func(c context.T, event *event.T) (reject bool, msg string) {
		for _, tag := range event.Tags {
			if len(tag) > 1 && len(tag[0]) == 1 {
				if len(tag[1]) > maxTagValueLen {
					return true, "event contains too large tags"
				}
			}
		}
		return false, ""
	}
}

// RestrictToSpecifiedKinds returns a function that can be used as a
// RejectFilter that will reject any events with kinds different than the
// specified ones.
func RestrictToSpecifiedKinds(kinds ...kind.T) func(context.T, *event.T) (bool, string) {
	var kMax, kMin kind.T
	for _, kind := range kinds {
		if kind > kMax {
			kMax = kind
		}
		if kind < kMin {
			kMin = kind
		}
	}
	return func(c context.T, event *event.T) (reject bool, msg string) {
		// these are cheap and very questionable optimizations, but they exist for a reason:
		// we would have to ensure that the kind number is within the bounds of a uint16 anyway
		if event.Kind > kMax {
			return true, "event kind not allowed"
		}
		if event.Kind < kMin {
			return true, "event kind not allowed"
		}
		// hopefully this map of uint16s is very fast
		if _, allowed := slices.BinarySearch(kinds, event.Kind); allowed {
			return false, ""
		}
		return true, "event kind not allowed"
	}
}

func PreventTimestampsInThePast(thresholdSeconds timestamp.T) func(context.T, *event.T) (bool, string) {
	return func(c context.T, event *event.T) (reject bool, msg string) {
		if timestamp.Now()-event.CreatedAt > thresholdSeconds {
			return true, "event too old"
		}
		return false, ""
	}
}

func PreventTimestampsInTheFuture(thresholdSeconds timestamp.T) func(context.T, *event.T) (bool, string) {
	return func(c context.T, event *event.T) (reject bool, msg string) {
		if event.CreatedAt-timestamp.Now() > thresholdSeconds {
			return true, "event too much in the future"
		}
		return false, ""
	}
}
