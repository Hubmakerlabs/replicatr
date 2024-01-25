package replicatr

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"golang.org/x/exp/slices"
)

// NoComplexFilters disallows filters with more than 2 tags.
func NoComplexFilters(c context.T, f *filter.T) (rej bool, msg string) {
	items := len(f.Tags) + len(f.Kinds)
	if items > 4 && len(f.Tags) > 2 {
		return true, "too many things to filter for"
	}
	return false, ""
}

// NoEmptyFilters disallows filters that don't have at least a tag, a kind, an
// author or an id.
func NoEmptyFilters(c context.T, f *filter.T) (reject bool, msg string) {
	cf := len(f.Kinds) + len(f.IDs) + len(f.Authors)
	for _, tagItems := range f.Tags {
		cf += len(tagItems)
	}
	if cf == 0 {
		return true, "can't handle empty filters"
	}
	return false, ""
}

// AntiSyncBots tries to prevent people from syncing kind:1s from this relay to
// else by always requiring an author parameter at least.
func AntiSyncBots(c context.T, f *filter.T) (rej bool, msg string) {
	return (len(f.Kinds) == 0 ||
			slices.Contains(f.Kinds, 1)) &&
			len(f.Authors) == 0,
		"an author must be specified to get their kind:1 notes"
}

func NoSearchQueries(c context.T, f *filter.T) (reject bool, msg string) {
	if f.Search != "" {
		return true, "search is not supported"
	}
	return false, ""
}

func RemoveSearchQueries(c context.T, f *filter.T) {
	f.Search = ""
}

func RemoveAllButKinds(k ...kind.T) OverwriteFilter {
	return func(c context.T, f *filter.T) {
		if n := len(f.Kinds); n > 0 {
			newKinds := make(kinds.T, 0, n)
			for i := 0; i < n; i++ {
				if kk := f.Kinds[i]; slices.Contains(k, kk) {
					newKinds = append(newKinds, kk)
				}
			}
			f.Kinds = newKinds
		}
	}
}

func RemoveAllButTags(tagNames ...string) OverwriteFilter {
	return func(c context.T, f *filter.T) {
		for tagName := range f.Tags {
			if !slices.Contains(tagNames, tagName) {
				delete(f.Tags, tagName)
			}
		}
	}
}
