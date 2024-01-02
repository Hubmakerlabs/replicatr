package sdk

import (
	"context"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr"
)

func (s *System) ExpandQueriesByAuthorAndRelays(
	ctx context.Context,
	filter *nip1.Filter,
) (filters map[*nostr.Relay]*nip1.Filter, e error) {

	n := len(filter.Authors)
	if n == 0 {
		return nil, fmt.Errorf("no authors in filter")
	}
	relaysForPubkey := make(map[string][]*nostr.Relay, n)
	wg := sync.WaitGroup{}
	wg.Add(n)
	for _, pubkey := range filter.Authors {
		go func(pubkey string) {
			defer wg.Done()
			relayURLs := s.FetchOutboxRelays(ctx, pubkey)
			c := 0
			for _, r := range relayURLs {
				relay, err := s.Pool.EnsureRelay(r)
				if err != nil {
					continue
				}
				relaysForPubkey[pubkey] = append(relaysForPubkey[pubkey], relay)
				c++
				if c == 3 {
					return
				}
			}
		}(pubkey)
	}
	wg.Wait()
	filters = make(map[*nostr.Relay]*nip1.Filter, n) // { [relay]: filter }
	for pubkey, relays := range relaysForPubkey {
		for _, relay := range relays {
			flt, ok := filters[relay]
			if !ok {
				flt = filter.Clone()
				filters[relay] = flt
			}
			flt.Authors = append(flt.Authors, pubkey)
		}
	}
	return filters, nil
}
