package filter

import (
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/mailru/easyjson"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

type T struct {
	IDs     []string             `json:"ids,omitempty"`
	Kinds   []int                `json:"kinds,omitempty"`
	Authors []string             `json:"authors,omitempty"`
	Tags    TagMap               `json:"-,omitempty"`
	Since   *timestamp.Timestamp `json:"since,omitempty"`
	Until   *timestamp.Timestamp `json:"until,omitempty"`
	Limit   int                  `json:"limit,omitempty"`
	Search  string               `json:"search,omitempty"`
}

type TagMap map[string][]string

func (ef T) String() string {
	j, _ := easyjson.Marshal(ef)
	return string(j)
}

func (ef T) Matches(evt *event.T) bool {
	if evt == nil {
		return false
	}

	if ef.IDs != nil && !slices.Contains(ef.IDs, evt.ID) {
		return false
	}

	if ef.Kinds != nil && !slices.Contains(ef.Kinds, evt.Kind) {
		return false
	}

	if ef.Authors != nil && !slices.Contains(ef.Authors, evt.PubKey) {
		return false
	}

	for f, v := range ef.Tags {
		if v != nil && !evt.Tags.ContainsAny(f, v) {
			return false
		}
	}

	if ef.Since != nil && evt.CreatedAt < *ef.Since {
		return false
	}

	if ef.Until != nil && evt.CreatedAt > *ef.Until {
		return false
	}

	return true
}

func FilterEqual(a T, b T) bool {
	if !similar(a.Kinds, b.Kinds) {
		return false
	}

	if !similar(a.IDs, b.IDs) {
		return false
	}

	if !similar(a.Authors, b.Authors) {
		return false
	}

	if len(a.Tags) != len(b.Tags) {
		return false
	}

	for f, av := range a.Tags {
		if bv, ok := b.Tags[f]; !ok {
			return false
		} else {
			if !similar(av, bv) {
				return false
			}
		}
	}

	if !arePointerValuesEqual(a.Since, b.Since) {
		return false
	}

	if !arePointerValuesEqual(a.Until, b.Until) {
		return false
	}

	if a.Search != b.Search {
		return false
	}

	return true
}

func (ef T) Clone() T {
	clone := T{
		IDs:     slices.Clone(ef.IDs),
		Authors: slices.Clone(ef.Authors),
		Kinds:   slices.Clone(ef.Kinds),
		Limit:   ef.Limit,
		Search:  ef.Search,
	}

	if ef.Tags != nil {
		clone.Tags = make(TagMap, len(ef.Tags))
		for k, v := range ef.Tags {
			clone.Tags[k] = slices.Clone(v)
		}
	}

	if ef.Since != nil {
		since := *ef.Since
		clone.Since = &since
	}

	if ef.Until != nil {
		until := *ef.Until
		clone.Until = &until
	}

	return clone
}

func arePointerValuesEqual[V comparable](a *V, b *V) bool {
	if a == nil && b == nil {
		return true
	}
	if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func similar[E constraints.Ordered](as, bs []E) bool {
	if len(as) != len(bs) {
		return false
	}

	for _, a := range as {
		for _, b := range bs {
			if b == a {
				goto next
			}
		}
		// didn't find a B that corresponded to the current A
		return false

	next:
		continue
	}

	return true
}
