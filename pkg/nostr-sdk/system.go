package sdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	filters2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/pools"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr-sdk/cache"
)

type System struct {
	RelaysCache      cache.Cache32[[]Relay]
	FollowsCache     cache.Cache32[[]Follow]
	MetadataCache    cache.Cache32[ProfileMetadata]
	Pool             *pools.SimplePool
	RelayListRelays  []string
	FollowListRelays []string
	MetadataRelays   []string
	Store            eventstore.Store
}

func (sys System) StoreRelay() eventstore.RelayInterface {
	return eventstore.RelayWrapper{Store: sys.Store}
}

func (sys System) FetchRelays(ctx context.Context, pubkey string) []Relay {
	if v, ok := sys.RelaysCache.Get(pubkey); ok {
		return v
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	res := FetchRelaysForPubkey(ctx, sys.Pool, pubkey, sys.RelayListRelays...)
	sys.RelaysCache.SetWithTTL(pubkey, res, time.Hour*6)
	return res
}

func (sys System) FetchOutboxRelays(ctx context.Context, pubkey string) []string {
	relays := sys.FetchRelays(ctx, pubkey)
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
func (sys System) FetchProfileMetadata(ctx context.Context, pubkey string) ProfileMetadata {
	pm, _ := sys.fetchProfileMetadata(ctx, pubkey)
	return pm
}

// FetchOrStoreProfileMetadata is like FetchProfileMetadata, but also saves the result to the sys.Store
func (sys System) FetchOrStoreProfileMetadata(ctx context.Context, pubkey string) ProfileMetadata {
	pm, fromInternal := sys.fetchProfileMetadata(ctx, pubkey)
	if !fromInternal {
		sys.StoreRelay().Publish(ctx, pm.Event)
	}
	return pm
}

func (sys System) fetchProfileMetadata(ctx context.Context, pubkey string) (pm ProfileMetadata, fromInternal bool) {
	if v, ok := sys.MetadataCache.Get(pubkey); ok {
		return v, true
	}

	if sys.Store != nil {
		res, _ := sys.StoreRelay().QuerySync(ctx, &filter.T{Kinds: []int{0}, Authors: []string{pubkey}})
		if len(res) != 0 {
			if m, err := ParseMetadata(res[0]); err == nil {
				m.PubKey = pubkey
				m.Event = res[0]
				sys.MetadataCache.SetWithTTL(pubkey, m, time.Hour*6)
				return m, true
			}
		}
	}

	ctxRelays, cancel := context.WithTimeout(ctx, time.Second*2)
	relays := sys.FetchOutboxRelays(ctxRelays, pubkey)
	cancel()

	ctx, cancel = context.WithTimeout(ctx, time.Second*3)
	res := FetchProfileMetadata(ctx, sys.Pool, pubkey, append(relays, sys.MetadataRelays...)...)
	cancel()

	sys.MetadataCache.SetWithTTL(pubkey, res, time.Hour*6)
	return res, false
}

// FetchUserEvents fetches events from each users' outbox relays, grouping queries when possible.
func (sys System) FetchUserEvents(ctx context.Context, filt filter.T) (map[string][]*event.T, error) {
	filters, err := sys.ExpandQueriesByAuthorAndRelays(ctx, filt)
	if err != nil {
		return nil, fmt.Errorf("failed to expand queries: %w", err)
	}

	results := make(map[string][]*event.T)
	wg := sync.WaitGroup{}
	wg.Add(len(filters))
	for rl, ff := range filters {
		go func(rl *relays.Relay, f filter.T) {
			defer wg.Done()
			f.Limit = f.Limit * len(f.Authors) // hack
			sub, err := rl.Subscribe(ctx, filters2.T{filt})
			if err != nil {
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
