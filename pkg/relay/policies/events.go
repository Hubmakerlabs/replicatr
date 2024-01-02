package policies

import (
	"context"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"golang.org/x/exp/slices"
)

// PreventTooManyIndexableTags returns a function that can be used as a RejectFilter that will reject
// events with more indexable (single-character) tags than the specified number.
//
// If ignoreKinds is given this restriction will not apply to these kinds (useful for allowing a bigger).
// If onlyKinds is given then all other kinds will be ignored.
func PreventTooManyIndexableTags(max int, ignoreKinds kinds.T, onlyKinds kinds.T) func(context.Context, *nip1.Event) (bool, string) {
	ignore := func(kind kind.T) bool { return false }
	if len(ignoreKinds) > 0 {
		ignore = func(kind kind.T) bool {
			_, isIgnored := slices.BinarySearch(ignoreKinds, kind)
			return isIgnored
		}
	}
	if len(onlyKinds) > 0 {
		ignore = func(kind kind.T) bool {
			_, isApplicable := slices.BinarySearch(onlyKinds, kind)
			return !isApplicable
		}
	}

	return func(ctx context.Context, event *nip1.Event) (reject bool, msg string) {
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

// PreventLargeTags rejects events that have indexable tag values greater than maxTagValueLen.
func PreventLargeTags(maxTagValueLen int) func(context.Context, *nip1.Event) (bool, string) {
	return func(ctx context.Context, event *nip1.Event) (reject bool, msg string) {
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

// RestrictToSpecifiedKinds returns a function that can be used as a RejectFilter that will reject
// any events with kinds different than the specified ones.
//
// todo: this range minimum doesn't look like it would ever change from zero
func RestrictToSpecifiedKinds(kinds ...kind.T) func(context.Context, *nip1.Event) (bool, string) {
	var maximum, minimum kind.T
	for _, kind := range kinds {
		if kind > maximum {
			maximum = kind
		}

		if kind < minimum {
			minimum = kind
		}
	}

	return func(ctx context.Context, event *nip1.Event) (reject bool, msg string) {
		// these are cheap and very questionable optimizations, but they exist for a reason:
		// we would have to ensure that the kind number is within the bounds of a uint16 anyway
		if event.Kind > maximum {
			return true, "event kind not allowed"
		}
		if event.Kind < minimum {
			return true, "event kind not allowed"
		}

		// hopefully this map of uint16s is very fast
		if _, allowed := slices.BinarySearch(kinds, event.Kind); allowed {
			return false, ""
		}
		return true, "event kind not allowed"
	}
}

func PreventTimestampsInThePast(thresholdSeconds timestamp.T) func(context.Context, *nip1.Event) (bool, string) {
	return func(ctx context.Context, event *nip1.Event) (reject bool, msg string) {
		if timestamp.Now()-event.CreatedAt > thresholdSeconds {
			return true, "event too old"
		}
		return false, ""
	}
}

func PreventTimestampsInTheFuture(thresholdSeconds timestamp.T) func(context.Context, *nip1.Event) (bool, string) {
	return func(ctx context.Context, event *nip1.Event) (reject bool, msg string) {
		if event.CreatedAt-timestamp.Now() > thresholdSeconds {
			return true, "event too much in the future"
		}
		return false, ""
	}
}
