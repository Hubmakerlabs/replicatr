package sdk

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/pool"
)

type Relay struct {
	URL    string
	Inbox  bool
	Outbox bool
}

func FetchRelaysForPubkey(ctx context.T, pool *pool.SimplePool, pubkey string, relays ...string) (r []Relay) {
	ctx, cancel := context.Cancel(ctx)
	defer cancel()
	ch := pool.SubManyEose(ctx, relays, filters.T{
		{
			Kinds: kinds.T{
				kind.RelayListMetadata,
				kind.ContactList,
			},
			Authors: []string{pubkey},
			Limit:   2,
		},
	})
	r = make([]Relay, 0, 20)
	i := 0
	for ie := range ch {
		switch ie.T.Kind {
		case kind.RelayListMetadata:
			r = append(r, ParseRelaysFromKind10002(ie.T)...)
		case kind.ContactList:
			r = append(r, ParseRelaysFromKind3(ie.T)...)
		}
		i++
		if i >= 2 {
			break
		}
	}
	return
}

func ParseRelaysFromKind10002(evt *event.T) (r []Relay) {
	r = make([]Relay, 0, len(evt.Tags))
	for _, tag := range evt.Tags {
		if u := tag.Value(); u != "" && tag[0] == "r" {
			if !IsValidRelayURL(u) {
				continue
			}
			rl := Relay{
				URL: normalize.URL(u),
			}
			if len(tag) == 2 {
				rl.Inbox = true
				rl.Outbox = true
			} else if tag[2] == "write" {
				rl.Outbox = true
			} else if tag[2] == "read" {
				rl.Inbox = true
			}
			r = append(r, rl)
		}
	}
	return
}

func ParseRelaysFromKind3(evt *event.T) (r []Relay) {
	type Item struct {
		Read  bool `json:"read"`
		Write bool `json:"write"`
	}
	items := make(map[string]Item, 20)
	var e error
	if e = json.Unmarshal([]byte(evt.Content), &items); log.Fail(e) {
		// shouldn't this be fatal?
	}
	r = make([]Relay, len(items))
	i := 0
	for u, item := range items {
		if !IsValidRelayURL(u) {
			continue
		}
		rl := Relay{
			URL: normalize.URL(u),
		}
		if item.Read {
			rl.Inbox = true
		}
		if item.Write {
			rl.Outbox = true
		}
		r = append(r, rl)
		i++
	}
	return r
}

func IsValidRelayURL(u string) bool {
	parsed, e := url.Parse(u)
	if log.Fail(e) {
		return false
	}
	if parsed.Scheme != "wss" && parsed.Scheme != "ws" {
		return false
	}
	if len(strings.Split(parsed.Host, ".")) < 2 {
		return false
	}
	return true
}
