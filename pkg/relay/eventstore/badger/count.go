package badger

import (
	"encoding/binary"
	"errors"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	nostr_binary "github.com/Hubmakerlabs/replicatr/pkg/nostr/binary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/dgraph-io/badger/v4"
)

func (b *Backend) CountEvents(c context.T, f *filter.T) (cnt int64, e error) {
	var queries []query
	var extraFilter *filter.T
	var since uint32
	if queries, extraFilter, since, e = prepareQueries(f); log.E.Chk(e) {
		return
	}
	e = b.View(func(txn *badger.Txn) (e error) {
		// iterate only through keys and in reverse order
		opts := badger.IteratorOptions{
			Reverse: true,
		}
		// actually iterate
		for _, q := range queries {
			it := txn.NewIterator(opts)
			defer it.Close()

			for it.Seek(q.startingPoint); it.ValidForPrefix(q.prefix); it.Next() {
				item := it.Item()
				key := item.Key()
				idxOffset := len(key) - 4 // this is where the idx actually starts
				// "id" indexes don't contain a timestamp
				if !q.skipTimestamp {
					createdAt := binary.BigEndian.Uint32(key[idxOffset-4 : idxOffset])
					if createdAt < since {
						break
					}
				}
				idx := make([]byte, 5)
				idx[0] = rawEventStorePrefix
				copy(idx[1:], key[idxOffset:])
				if extraFilter == nil {
					cnt++
				} else {
					// fetch actual event
					if item, e = txn.Get(idx); log.E.Chk(e) {
						if errors.Is(e, badger.ErrDiscardedTxn) {
							return
						}
						log.E.F("badger: count (%v) failed to get %d from raw event store: %s\n", q, idx, e)
						return
					}
					e = item.Value(func(val []byte) (e error) {
						evt := &event.T{}
						if e = nostr_binary.Unmarshal(val, evt); log.E.Chk(e) {
							return e
						}
						// check if this matches the other filters that were not part of the index
						if extraFilter == nil || extraFilter.Matches(evt) {
							cnt++
						}
						return nil
					})
					if log.E.Chk(e) {
						log.E.F("badger: count value read error: %s\n", e)
					}
				}
			}
		}
		return nil
	})
	return
}
