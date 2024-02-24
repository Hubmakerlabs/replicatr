package main

import (
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/client"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/kinds"
)

func (cfg *C) GetRelaysAndTags(pub string, m *Checklist) RelayIter {
	return func(c context.T, rl *client.T) bool {
		evs, err := rl.QuerySync(c, &filter.T{
			Kinds:   kinds.T{kind.FollowList},
			Authors: []string{pub},
			Limit:   &one,
		})
		if chk.D(err) {
			return true
		}
		// log.D.S(evs)
		for _, ev := range evs {
			log.D.S(ev.Tags)
			for _, tag := range ev.Tags {
				if len(tag) >= 2 && tag[0] == "p" {
					log.D.Ln("p tag", tag.Relay(), tag.Key(), tag.Value())
					cfg.Lock()
					(*m)[tag[1]] = struct{}{}
					cfg.Unlock()
				}
			}
			// todo: this breaks the relay list so don't do it, must be some other
			//  reason for it (getting relay lists?)
			//
			// if cfg.tempRelay == false {
			// 	log.D.Ln(ev.Content)
			// 	var rm Relays
			// 	if err = json.Unmarshal([]byte(ev.Content), &rm); chk.D(err) {
			// 		// continue
			// 	} else {
			// 		for k, v1 := range cfg.Relays {
			// 			if v2, ok := rm[k]; ok {
			// 				v2.Search = v1.Search
			// 			}
			// 		}
			// 		// cfg.Relays[rm]
			// 	}
			// }
			// cfg.Lock()
			// log.D.S(*m)
			// cfg.Unlock()
		}
		return true
	}
}
