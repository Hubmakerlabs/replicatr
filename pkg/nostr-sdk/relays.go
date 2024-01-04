package sdk

import (
	"context"
	"encoding/json"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/pools"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
)

type Relay struct {
	URL    string
	Inbox  bool
	Outbox bool
}

func FetchRelaysForPubkey(ctx context.Context, pool *pools.SimplePool, pubkey string, relays ...string) []Relay {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := pool.SubManyEose(ctx, relays, filter.Filters{
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
			if !relays.IsValidRelayURL(u) {
				continue
			}
			u := normalize.URL(u)

			relay := Relay{
				URL: u,
			}

			if len(tag) == 2 {
				relay.Inbox = true
				relay.Outbox = true
			} else if tag[2] == "write" {
				relay.Outbox = true
			} else if tag[2] == "read" {
				relay.Inbox = true
			}

			result = append(result, relay)
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
		if !relays.IsValidRelayURL(u) {
			continue
		}
		u := normalize.URL(u)

		relay := Relay{
			URL: u,
		}

		if item.Read {
			relay.Inbox = true
		}
		if item.Write {
			relay.Outbox = true
		}

		results = append(results, relay)
		i++
	}

	return results
}
