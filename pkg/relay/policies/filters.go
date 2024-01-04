package policies

import (
	"context"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	kinds2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"

	"golang.org/x/exp/slices"
)

// NoComplexFilters disallows filters with more than 2 tags.
func NoComplexFilters(ctx context.Context, f filter.T) (reject bool, msg string) {
	items := len(f.Tags) + len(f.Kinds)

	if items > 4 && len(f.Tags) > 2 {
		return true, "too many things to filter for"
	}

	return false, ""
}

// NoEmptyFilters disallows filters that don't have at least a tag, a kind, an author or an id.
func NoEmptyFilters(ctx context.Context, f filter.T) (reject bool, msg string) {
	c := len(f.Kinds) + len(f.IDs) + len(f.Authors)
	for _, tagItems := range f.Tags {
		c += len(tagItems)
	}
	if c == 0 {
		return true, "can't handle empty filters"
	}
	return false, ""
}

// AntiSyncBots tries to prevent people from syncing kind:1s from this relay to else by always
// requiring an author parameter at least.
func AntiSyncBots(ctx context.Context, f filter.T) (reject bool, msg string) {
	return (len(f.Kinds) == 0 || slices.Contains(f.Kinds, 1)) &&
		len(f.Authors) == 0, "an author must be specified to get their kind:1 notes"
}

func NoSearchQueries(ctx context.Context, f filter.T) (reject bool, msg string) {
	if f.Search != "" {
		return true, "search is not supported"
	}
	return false, ""
}

func RemoveSearchQueries(ctx context.Context, f *filter.T) {
	f.Search = ""
}

func RemoveAllButKinds(kinds ...uint16) func(context.Context, *filter.T) {
	return func(ctx context.Context, f *filter.T) {
		if n := len(f.Kinds); n > 0 {
			newKinds := make(kinds2.T, 0, n)
			for i := 0; i < n; i++ {
				if k := f.Kinds[i]; slices.Contains(kinds, uint16(k)) {
					newKinds = append(newKinds, k)
				}
			}
			f.Kinds = newKinds
		}
	}
}

func RemoveAllButTags(tagNames ...string) func(context.Context, *filter.T) {
	return func(ctx context.Context, f *filter.T) {
		for tagName := range f.Tags {
			if !slices.Contains(tagNames, tagName) {
				delete(f.Tags, tagName)
			}
		}
	}
}
