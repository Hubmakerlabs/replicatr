package sdk

import (
	"fmt"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
)

func (s *System) ExpandQueriesByAuthorAndRelays(
	ctx context.T,
	f *filter.T,
) (filters map[*relay.Relay]*filter.T, e error) {

	n := len(f.Authors)
	if n == 0 {
		return nil, fmt.Errorf("no authors in filter")
	}
	relaysForPubkey := make(map[string][]*relay.Relay, n)
	wg := sync.WaitGroup{}
	wg.Add(n)
	for _, pubkey := range f.Authors {
		go func(pubkey string) {
			defer wg.Done()
			relayURLs := s.FetchOutboxRelays(ctx, pubkey)
			c := 0
			for _, r := range relayURLs {
				var rl *relay.Relay
				if rl, e = s.Pool.EnsureRelay(r); log.E.Chk(e) {
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
	filters = make(map[*relay.Relay]*filter.T, n) // { [relay]: filter }
	for pubkey, relays := range relaysForPubkey {
		for _, rl := range relays {
			flt, ok := filters[rl]
			if !ok {
				flt = f.Clone()
				filters[rl] = flt
			}
			flt.Authors = append(flt.Authors, pubkey)
		}
	}
	return filters, nil
}
