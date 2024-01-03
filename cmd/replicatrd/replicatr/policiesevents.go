package replicatr

import (
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr"
	"golang.org/x/exp/slices"
)

// PreventTooManyIndexableTags returns a function that can be used as a
// RejectFilter that will reject events with more indexable (single-character)
// tags than the specified number.
//
// If ignoreKinds is given this restriction will not apply to these kinds
// (useful for allowing a bigger). If onlyKinds is given then all other kinds
// will be ignored.
func PreventTooManyIndexableTags(max int, ignoreKinds []int,
	onlyKinds []int) func(Ctx, *Event) (bool, string) {

	ignore := func(kind int) bool { return false }
	if len(ignoreKinds) > 0 {
		ignore = func(kind int) bool {
			_, isIgnored := slices.BinarySearch(ignoreKinds, kind)
			return isIgnored
		}
	}
	if len(onlyKinds) > 0 {
		ignore = func(kind int) bool {
			_, isApplicable := slices.BinarySearch(onlyKinds, kind)
			return !isApplicable
		}
	}
	return func(ctx Ctx, event *Event) (reject bool, msg string) {
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
func PreventLargeTags(maxTagValueLen int) func(Ctx, *Event) (bool, string) {
	return func(ctx Ctx, event *Event) (reject bool, msg string) {
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
func RestrictToSpecifiedKinds(kinds ...uint16) func(Ctx, *Event) (bool, string) {
	kMax := 0
	kMin := 0
	for _, kind := range kinds {
		if int(kind) > kMax {
			kMax = int(kind)
		}
		if int(kind) < kMin {
			kMin = int(kind)
		}
	}
	return func(ctx Ctx, event *Event) (reject bool, msg string) {
		// these are cheap and very questionable optimizations, but they exist for a reason:
		// we would have to ensure that the kind number is within the bounds of a uint16 anyway
		if event.Kind > kMax {
			return true, "event kind not allowed"
		}
		if event.Kind < kMin {
			return true, "event kind not allowed"
		}
		// hopefully this map of uint16s is very fast
		if _, allowed := slices.BinarySearch(kinds, uint16(event.Kind)); allowed {
			return false, ""
		}
		return true, "event kind not allowed"
	}
}

func PreventTimestampsInThePast(thresholdSeconds Timestamp) func(Ctx, *Event) (bool, string) {
	return func(ctx Ctx, event *Event) (reject bool, msg string) {
		if nostr.Now()-event.CreatedAt > thresholdSeconds {
			return true, "event too old"
		}
		return false, ""
	}
}

func PreventTimestampsInTheFuture(thresholdSeconds Timestamp) func(Ctx, *Event) (bool, string) {
	return func(ctx Ctx, event *Event) (reject bool, msg string) {
		if event.CreatedAt-nostr.Now() > thresholdSeconds {
			return true, "event too much in the future"
		}
		return false, ""
	}
}
