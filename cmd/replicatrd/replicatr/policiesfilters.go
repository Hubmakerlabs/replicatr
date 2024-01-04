package replicatr

import (
	"context"

	"golang.org/x/exp/slices"
)

// NoComplexFilters disallows filters with more than 2 tags.
func NoComplexFilters(ctx Ctx, f *Filter) (reject bool, msg string) {
	items := len(f.Tags) + len(f.Kinds)
	if items > 4 && len(f.Tags) > 2 {
		return true, "too many things to filter for"
	}
	return false, ""
}

// NoEmptyFilters disallows filters that don't have at least a tag, a kind, an
// author or an id.
func NoEmptyFilters(ctx Ctx, f *Filter) (reject bool, msg string) {
	c := len(f.Kinds) + len(f.IDs) + len(f.Authors)
	for _, tagItems := range f.Tags {
		c += len(tagItems)
	}
	if c == 0 {
		return true, "can't handle empty filters"
	}
	return false, ""
}

// AntiSyncBots tries to prevent people from syncing kind:1s from this relay to
// else by always requiring an author parameter at least.
func AntiSyncBots(ctx Ctx, f *Filter) (reject bool, msg string) {
	return (len(f.Kinds) == 0 || slices.Contains(f.Kinds, 1)) &&
		len(f.Authors) == 0, "an author must be specified to get their kind:1 notes"
}

func NoSearchQueries(ctx context.Context, f *Filter) (reject bool, msg string) {
	if f.Search != "" {
		return true, "search is not supported"
	}
	return false, ""
}

func RemoveSearchQueries(ctx Ctx, f *Filter) {
	f.Search = ""
}

func RemoveAllButKinds(kinds ...uint16) func(Ctx, *Filter) {
	return func(ctx Ctx, f *Filter) {
		if n := len(f.Kinds); n > 0 {
			newKinds := make([]int, 0, n)
			for i := 0; i < n; i++ {
				if k := f.Kinds[i]; slices.Contains(kinds, uint16(k)) {
					newKinds = append(newKinds, k)
				}
			}
			f.Kinds = newKinds
		}
	}
}

func RemoveAllButTags(tagNames ...string) func(Ctx, *Filter) {
	return func(ctx Ctx, f *Filter) {
		for tagName := range f.Tags {
			if !slices.Contains(tagNames, tagName) {
				delete(f.Tags, tagName)
			}
		}
	}
}
