package sdk

import (
	"fmt"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/interfaces/cache32"

	"github.com/Hubmakerlabs/replicatr/pkg/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	filters2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/pools"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relay"
)

type System struct {
	RelaysCache      cache32.I[[]Relay]
	FollowsCache     cache32.I[[]Follow]
	MetadataCache    cache32.I[ProfileMetadata]
	Pool             *pools.SimplePool
	RelayListRelays  []string
	FollowListRelays []string
	MetadataRelays   []string
	Store            eventstore.Store
}

func (sys System) StoreRelay() eventstore.RelayInterface {
	return eventstore.RelayWrapper{Store: sys.Store}
}

func (sys System) FetchRelays(c context.T, pubkey string) []Relay {
	if v, ok := sys.RelaysCache.Get(pubkey); ok {
		return v
	}

	c, cancel := context.Timeout(c, time.Second*5)
	defer cancel()

	res := FetchRelaysForPubkey(c, sys.Pool, pubkey, sys.RelayListRelays...)
	sys.RelaysCache.SetWithTTL(pubkey, res, time.Hour*6)
	return res
}

func (sys System) FetchOutboxRelays(c context.T, pubkey string) []string {
	relays := sys.FetchRelays(c, pubkey)
	result := make([]string, 0, len(relays))
	for _, rl := range relays {
		if rl.Outbox {
			result = append(result, rl.URL)
		}
	}
	return result
}

// FetchProfileMetadata fetches metadata for a given user from the local cache, or from the local store,
// or, failing these, from the target user's defined outbox relays -- then caches the result.
func (sys System) FetchProfileMetadata(c context.T, pubkey string) ProfileMetadata {
	pm, _ := sys.fetchProfileMetadata(c, pubkey)
	return pm
}

// FetchOrStoreProfileMetadata is like FetchProfileMetadata, but also saves the result to the sys.Store
func (sys System) FetchOrStoreProfileMetadata(c context.T, pubkey string) ProfileMetadata {
	pm, fromInternal := sys.fetchProfileMetadata(c, pubkey)
	if !fromInternal {
		sys.StoreRelay().Publish(c, pm.Event)
	}
	return pm
}

func (sys System) fetchProfileMetadata(c context.T, pubkey string) (pm ProfileMetadata, fromInternal bool) {
	if v, ok := sys.MetadataCache.Get(pubkey); ok {
		return v, true
	}

	if sys.Store != nil {
		res, _ := sys.StoreRelay().QuerySync(c, &filter.T{Kinds: []int{0}, Authors: []string{pubkey}})
		if len(res) != 0 {
			if m, e := ParseMetadata(res[0]); e == nil {
				m.PubKey = pubkey
				m.Event = res[0]
				sys.MetadataCache.SetWithTTL(pubkey, m, time.Hour*6)
				return m, true
			}
		}
	}

	ctxRelays, cancel := context.Timeout(c, time.Second*2)
	relays := sys.FetchOutboxRelays(ctxRelays, pubkey)
	cancel()

	c, cancel = context.Timeout(c, time.Second*3)
	res := FetchProfileMetadata(c, sys.Pool, pubkey, append(relays, sys.MetadataRelays...)...)
	cancel()

	sys.MetadataCache.SetWithTTL(pubkey, res, time.Hour*6)
	return res, false
}

// FetchUserEvents fetches events from each users' outbox relays, grouping queries when possible.
func (sys System) FetchUserEvents(c context.T, filt filter.T) (map[string][]*event.T, error) {
	filters, e := sys.ExpandQueriesByAuthorAndRelays(c, filt)
	if e != nil {
		return nil, fmt.Errorf("failed to expand queries: %w", e)
	}

	results := make(map[string][]*event.T)
	wg := sync.WaitGroup{}
	wg.Add(len(filters))
	for rl, ff := range filters {
		go func(rl *relay.Relay, f filter.T) {
			defer wg.Done()
			f.Limit = f.Limit * len(f.Authors) // hack
			sub, e := rl.Subscribe(c, filters2.T{filt})
			if e != nil {
				return
			}
			for {
				select {
				case evt := <-sub.Events:
					results[evt.PubKey] = append(results[evt.PubKey], evt)
				case <-sub.EndOfStoredEvents:
					return
				}
			}
		}(rl, ff)
	}
	wg.Wait()

	return results, nil
}
