package sdk

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr"
)

type Relay struct {
	URL    string
	Inbox  bool
	Outbox bool
}

func FetchRelaysForPubkey(ctx context.Context, pool *nostr.SimplePool, pubkey string, relays ...string) (r []Relay) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch := pool.SubManyEose(ctx, relays, nip1.Filters{
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
		switch ie.Event.Kind {
		case kind.RelayListMetadata:
			r = append(r, ParseRelaysFromKind10002(ie.Event)...)
		case kind.ContactList:
			r = append(r, ParseRelaysFromKind3(ie.Event)...)
		}
		i++
		if i >= 2 {
			break
		}
	}
	return
}

func ParseRelaysFromKind10002(evt *nip1.Event) (r []Relay) {
	r = make([]Relay, 0, len(evt.Tags))
	for _, tag := range evt.Tags {
		if u := tag.Value(); u != "" && tag[0] == "r" {
			if !IsValidRelayURL(u) {
				continue
			}
			relay := Relay{
				URL: normalize.URL(u),
			}
			if len(tag) == 2 {
				relay.Inbox = true
				relay.Outbox = true
			} else if tag[2] == "write" {
				relay.Outbox = true
			} else if tag[2] == "read" {
				relay.Inbox = true
			}
			r = append(r, relay)
		}
	}
	return
}

func ParseRelaysFromKind3(evt *nip1.Event) (r []Relay) {
	type Item struct {
		Read  bool `json:"read"`
		Write bool `json:"write"`
	}
	items := make(map[string]Item, 20)
	var e error
	if e = json.Unmarshal([]byte(evt.Content), &items); fails(e) {
		// shouldn't this be fatal?
	}
	r = make([]Relay, len(items))
	i := 0
	for u, item := range items {
		if !IsValidRelayURL(u) {
			continue
		}
		relay := Relay{
			URL: normalize.URL(u),
		}
		if item.Read {
			relay.Inbox = true
		}
		if item.Write {
			relay.Outbox = true
		}
		r = append(r, relay)
		i++
	}
	return r
}

func IsValidRelayURL(u string) bool {
	parsed, e := url.Parse(u)
	if fails(e) {
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
