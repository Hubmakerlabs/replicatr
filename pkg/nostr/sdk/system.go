package sdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/sdk/cache"
	"github.com/Hubmakerlabs/replicatr/pkg/pool"
	"github.com/Hubmakerlabs/replicatr/pkg/relay/eventstore"
)

type System struct {
	RelaysCache      cache.Cache32[[]Relay]
	FollowsCache     cache.Cache32[[]Follow]
	MetadataCache    cache.Cache32[*ProfileMetadata]
	Pool             *pool.SimplePool
	RelayListRelays  []string
	FollowListRelays []string
	MetadataRelays   []string
	Store            eventstore.Store
}

func (s *System) StoreRelay() eventstore.RelayInterface {
	return eventstore.RelayWrapper{Store: s.Store}
}

func (s *System) FetchRelays(ctx context.Context, pubkey string) []Relay {
	if v, ok := s.RelaysCache.Get(pubkey); ok {
		return v
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	res := FetchRelaysForPubkey(ctx, s.Pool, pubkey, s.RelayListRelays...)
	s.RelaysCache.SetWithTTL(pubkey, res, time.Hour*6)
	return res
}

func (s *System) FetchOutboxRelays(ctx context.Context, pubkey string) []string {
	relays := s.FetchRelays(ctx, pubkey)
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
func (s *System) FetchProfileMetadata(ctx context.Context,
	pubkey string) *ProfileMetadata {

	pm, _ := s.fetchProfileMetadata(ctx, pubkey)
	return pm
}

// FetchOrStoreProfileMetadata is like FetchProfileMetadata, but also saves the
// result to the sys.Store
func (s *System) FetchOrStoreProfileMetadata(ctx context.Context,
	pubkey string) *ProfileMetadata {

	pm, fromInternal := s.fetchProfileMetadata(ctx, pubkey)
	if !fromInternal {
		s.StoreRelay().Publish(ctx, *pm.Event)
	}
	return pm
}

func (s *System) fetchProfileMetadata(ctx context.Context,
	pubkey string) (pm *ProfileMetadata, fromInternal bool) {

	if v, ok := s.MetadataCache.Get(pubkey); ok {
		return v, true
	}
	if s.Store != nil {
		res, e := s.StoreRelay().QuerySync(ctx, &filter.T{Kinds: kinds.T{kind.ProfileMetadata},
			Authors: []string{pubkey}})
		log.D.Chk(e)
		if len(res) != 0 {
			if pm, e = ParseMetadata(res[0]); !log.E.Chk(e) {
				pm.PubKey = pubkey
				pm.Event = res[0]
				s.MetadataCache.SetWithTTL(pubkey, pm, time.Hour*6)
				return pm, true
			}
		}
	}
	ctxRelays, cancel := context.WithTimeout(ctx, time.Second*2)
	relays := s.FetchOutboxRelays(ctxRelays, pubkey)
	cancel()
	ctx, cancel = context.WithTimeout(ctx, time.Second*3)
	res := FetchProfileMetadata(ctx, s.Pool, pubkey, append(relays, s.MetadataRelays...)...)
	cancel()
	s.MetadataCache.SetWithTTL(pubkey, res, time.Hour*6)
	return res, false
}

// FetchUserEvents fetches events from each users' outbox relays, grouping
// queries when possible.
func (s *System) FetchUserEvents(ctx context.Context,
	f *filter.T) (r map[string][]*event.T, e error) {

	var ff map[*relay.Relay]*filter.T
	if ff, e = s.ExpandQueriesByAuthorAndRelays(ctx,
		f); fails(e) {

		return nil, fmt.Errorf("failed to expand queries: %w", e)
	}
	r = make(map[string][]*event.T)
	wg := sync.WaitGroup{}
	wg.Add(len(ff))
	for rl, f := range ff {
		go func(rl *relay.Relay, f *filter.T) {
			defer wg.Done()
			f.Limit = f.Limit *
				len(f.Authors) // hack
			var sub *relay.Subscription
			if sub, e = rl.Subscribe(ctx,
				filters.T{f}); fails(e) {

				return
			}
			for {
				select {
				case evt := <-sub.Events:
					r[evt.PubKey] = append(r[evt.PubKey], evt)
				case <-sub.EndOfStoredEvents:
					return
				}
			}
		}(rl, f)
	}
	wg.Wait()
	return
}
