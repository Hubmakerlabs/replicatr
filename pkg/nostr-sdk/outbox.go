package sdk

import (
	"context"
	"fmt"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
)

func (sys *System) ExpandQueriesByAuthorAndRelays(
	ctx context.Context,
	f filter.Filter,
) (map[*relays.Relay]filter.Filter, error) {
	n := len(f.Authors)
	if n == 0 {
		return nil, fmt.Errorf("no authors in f")
	}

	relaysForPubkey := make(map[string][]*relays.Relay, n)

	wg := sync.WaitGroup{}
	wg.Add(n)
	for _, pubkey := range f.Authors {
		go func(pubkey string) {
			defer wg.Done()
			relayURLs := sys.FetchOutboxRelays(ctx, pubkey)
			c := 0
			for _, r := range relayURLs {
				relay, err := sys.Pool.EnsureRelay(r)
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

	filterForRelay := make(map[*relays.Relay]filter.Filter, n) // { [relay]: f }
	for pubkey, relays := range relaysForPubkey {
		for _, relay := range relays {
			flt, ok := filterForRelay[relay]
			if !ok {
				flt = f.Clone()
				filterForRelay[relay] = flt
			}
			flt.Authors = append(flt.Authors, pubkey)
		}
	}

	return filterForRelay, nil
}
