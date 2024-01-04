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
	f filter.T,
) (map[*relays.Relay]filter.T, error) {
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
				rl, e := sys.Pool.EnsureRelay(r)
				if e != nil {
					continue
				}
				relaysForPubkey[pubkey] = append(relaysForPubkey[pubkey], rl)
				c++
				if c == 3 {
					return
				}
			}
		}(pubkey)
	}
	wg.Wait()

	filterForRelay := make(map[*relays.Relay]filter.T, n) // { [relay]: f }
	for pubkey, relays := range relaysForPubkey {
		for _, rl := range relays {
			flt, ok := filterForRelay[rl]
			if !ok {
				flt = f.Clone()
				filterForRelay[rl] = flt
			}
			flt.Authors = append(flt.Authors, pubkey)
		}
	}

	return filterForRelay, nil
}
