package main

import (
	"encoding/json"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
)

func (cfg *C) PopulateFollows(f *[]string, start, end *int) RelayIter {
	return func(c context.T, rl *relay.T) bool {
		log.T.Ln("populating follow list of profile", rl.URL(), *f)
		evs, e := rl.QuerySync(c, &filter.T{
			Kinds:   kinds.T{kind.ProfileMetadata},
			Authors: (*f)[*start:*end], // Use the updated end index
			Limit:   *end - *start,
		})
		log.D.S(evs)
		if log.Fail(e) {
			return true
		}
		for _, ev := range evs {
			p := &Metadata{}
			e = json.Unmarshal([]byte(ev.Content), p)
			if e == nil {
				cfg.Lock()
				cfg.Follows[ev.PubKey] = p
				cfg.Unlock()
			}
		}
		return true
	}
}
