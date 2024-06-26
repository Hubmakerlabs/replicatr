package app

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"golang.org/x/exp/slices"
)

// NoComplexFilters disallows filters with more than 3 tags or total of 6 of kinds and tags in sum..
func NoComplexFilters(c context.T, id subscriptionid.T, f *filter.T) (rej bool, msg string) {
	items := len(f.Tags) + len(f.Kinds)
	if items > 6 && len(f.Tags) > 3 {
		return true, "too many things to filter for"
	}
	return false, ""
}

// NoEmptyFilters disallows filters that don't have at least a tag, a kind, an
// author or an id, or since or until.
func NoEmptyFilters(c context.T, id subscriptionid.T, f *filter.T) (reject bool, msg string) {
	cf := len(f.Kinds) + len(f.IDs) + len(f.Authors)
	for _, tagItems := range f.Tags {
		cf += len(tagItems)
	}
	if f.Since != nil {
		cf++
	}
	if f.Until != nil {
		cf++
	}
	if f.Limit != nil {
		cf++
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

func NoSearchQueries(c context.T, id subscriptionid.T,
	f *filter.T) (reject bool, msg string) {
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

func LimitAuthorsAndIDs(authors, ids int) OverwriteFilter {
	return func(c context.T, f *filter.T) {
		if len(f.Authors) > authors {
			log.I.Ln("limiting authors to", authors)
			f.Authors = f.Authors[:20]
		}
		if len(f.IDs) > ids {
			log.I.Ln("limiting IDs to", ids)
			f.IDs = f.IDs[:20]
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
