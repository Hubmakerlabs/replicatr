package filters

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
)

type T []*filter.T

func (eff T) ToArray() (a array.T) {
	for i := range eff {
		a = append(a, eff[i].ToObject())
	}
	return
}

func (eff T) String() string { return eff.ToArray().String() }

func (eff T) Match(event *event.T) bool {
	for _, f := range eff {
		if f.Matches(event) {
			return true
		}
	}
	return false
}
