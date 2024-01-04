package filters

import (
	"encoding/json"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
)

type T []filter.T

func (eff T) String() string {
	j, _ := json.Marshal(eff)
	return string(j)
}

func (eff T) Match(evt *event.T) bool {
	for _, f := range eff {
		if f.Matches(evt) {
			return true
		}
	}
	return false
}
