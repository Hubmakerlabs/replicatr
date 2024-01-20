package sdk

import (
	"encoding/json"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/pools"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
)

type Relay struct {
	URL    string
	Inbox  bool
	Outbox bool
}

func FetchRelaysForPubkey(c context.T, pool *pools.SimplePool, pubkey string, relays ...string) []Relay {
	c, cancel := context.Cancel(c)
	defer cancel()

	ch := pool.SubManyEose(c, relays, filters.T{
		{
			Kinds: []int{
				event.KindRelayListMetadata,
				event.KindContactList,
			},
			Authors: []string{pubkey},
			Limit:   2,
		},
	})

	result := make([]Relay, 0, 20)
	i := 0
	for ie := range ch {
		switch ie.T.Kind {
		case event.KindRelayListMetadata:
			result = append(result, ParseRelaysFromKind10002(ie.T)...)
		case event.KindContactList:
			result = append(result, ParseRelaysFromKind3(ie.T)...)
		}

		i++
		if i >= 2 {
			break
		}
	}

	return result
}

func ParseRelaysFromKind10002(evt *event.T) []Relay {
	result := make([]Relay, 0, len(evt.Tags))
	for _, tag := range evt.Tags {
		if u := tag.Value(); u != "" && tag[0] == "r" {
			if !relay.IsValidRelayURL(u) {
				continue
			}
			u := normalize.URL(u)

			rl := Relay{
				URL: u,
			}

			if len(tag) == 2 {
				rl.Inbox = true
				rl.Outbox = true
			} else if tag[2] == "write" {
				rl.Outbox = true
			} else if tag[2] == "read" {
				rl.Inbox = true
			}

			result = append(result, rl)
		}
	}

	return result
}

func ParseRelaysFromKind3(evt *event.T) []Relay {
	type Item struct {
		Read  bool `json:"read"`
		Write bool `json:"write"`
	}

	items := make(map[string]Item, 20)
	json.Unmarshal([]byte(evt.Content), &items)

	results := make([]Relay, len(items))
	i := 0
	for u, item := range items {
		if !relay.IsValidRelayURL(u) {
			continue
		}
		u := normalize.URL(u)

		rl := Relay{
			URL: u,
		}

		if item.Read {
			rl.Inbox = true
		}
		if item.Write {
			rl.Outbox = true
		}

		results = append(results, rl)
		i++
	}

	return results
}
