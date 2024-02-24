package app

import (
	"golang.org/x/exp/slices"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/kinds"
	"mleku.dev/git/nostr/timestamp"
)

// PreventExcessTags returns a function that can be used as a
// RejectFilter that will reject events with more indexable (single-character)
// tags than the specified number.
//
// If ignoreKinds is given this restriction will not apply to these kinds
// (useful for allowing a bigger). If onlyKinds is given then all other kinds
// will be ignored.
func PreventExcessTags(max int, ign kinds.T, only kinds.T) RejectEvent {
	ignore := func(kind kind.T) bool { return false }
	if len(ign) > 0 {
		ignore = func(k kind.T) bool {
			_, isIgnored := slices.BinarySearch(ign, k)
			return isIgnored
		}
	}
	if len(only) > 0 {
		ignore = func(k kind.T) bool {
			_, isApplicable := slices.BinarySearch(only, k)
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
func PreventLargeTags(maxTagValueLen int) RejectEvent {
	return func(c context.T, ev *event.T) (rej bool, msg string) {
		for _, tag := range ev.Tags {
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
func RestrictToSpecifiedKinds(kinds ...kind.T) RejectEvent {
	var kMax, kMin kind.T
	for _, kind := range kinds {
		if kind > kMax {
			kMax = kind
		}
		if kind < kMin {
			kMin = kind
		}
	}
	return func(c context.T, ev *event.T) (rej bool, msg string) {
		// these are cheap and very questionable optimizations, but they exist
		// for a reason: we would have to ensure that the kind number is within
		// the bounds of a uint16 anyway
		if ev.Kind > kMax {
			return true, "event kind not allowed"
		}
		if ev.Kind < kMin {
			return true, "event kind not allowed"
		}
		// hopefully this map of uint16s is very fast
		if _, allowed := slices.BinarySearch(kinds, ev.Kind); allowed {
			return false, ""
		}
		return true, "event kind not allowed"
	}
}

func PreventTimestampsInThePast(thresholdSeconds timestamp.T) RejectEvent {
	return func(c context.T, event *event.T) (reject bool, msg string) {
		if timestamp.Now()-event.CreatedAt > thresholdSeconds {
			return true, "event too old"
		}
		return false, ""
	}
}

func PreventTimestampsInTheFuture(thresholdSeconds timestamp.T) RejectEvent {
	return func(c context.T, event *event.T) (reject bool, msg string) {
		if event.CreatedAt-timestamp.Now() > thresholdSeconds {
			return true, "event too much in the future"
		}
		return false, ""
	}
}
