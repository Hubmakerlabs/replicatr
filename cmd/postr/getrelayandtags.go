package main

import (
	"encoding/json"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
)

func (cfg *C) GetRelaysAndTags(pub string, m *Checklist) RelayIter {
	return func(c context.T, rl *relay.Relay) bool {
		evs, e := rl.QuerySync(c, &filter.T{
			Kinds:   kinds.T{kind.FollowList},
			Authors: []string{pub},
			Limit:   1,
		})
		if log.Fail(e) {
			return true
		}
		log.D.Ln("evs", len(evs[0].Tags))
		for _, ev := range evs {
			var rm Relays
			if cfg.tempRelay == false {
				if e = json.Unmarshal([]byte(ev.Content), &rm); log.Fail(e) {
					// continue
				} else {
					for k, v1 := range cfg.Relays {
						if v2, ok := rm[k]; ok {
							v2.Search = v1.Search
						}
					}
					cfg.Relays = rm
				}
			}
			log.T.S(ev.Tags)
			for _, tag := range ev.Tags {
				if len(tag) >= 2 && tag[0] == "p" {
					log.T.Ln("p tag", tag.Relay(), tag.Key(), tag.Value())
					cfg.Lock()
					(*m)[tag[1]] = struct{}{}
					cfg.Unlock()
				}
			}
			// cfg.Lock()
			// log.D.S(*m)
			// cfg.Unlock()
		}
		return true
	}
}
