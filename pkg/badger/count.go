package badger

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	nostr_binary "github.com/Hubmakerlabs/replicatr/pkg/nostr/binary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/dgraph-io/badger/v4"
)

var serialDelete uint32 = 0

func (b *BadgerBackend) CountEvents(ctx context.Context,
	filter nip1.Filter) (int64, error) {
	var count int64 = 0

	queries, extraFilter, since, err := prepareQueries(filter)
	if err != nil {
		return 0, err
	}

	err = b.View(func(txn *badger.Txn) error {
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
					count++
				} else {
					// fetch actual event
					item, err = txn.Get(idx)
					if err != nil {
						if errors.Is(badger.ErrDiscardedTxn, err) {
							return err
						}
						log.E.F("badger: count (%v) failed to get %d from raw event store: %s\n",
							q, idx, err)
						return err
					}

					err = item.Value(func(val []byte) error {
						var evt nip1.Event
						if err := nostr_binary.Unmarshal(val,
							&evt); err != nil {
							return err
						}

						// check if this matches the other filters that were not part of the index
						if extraFilter == nil || extraFilter.Matches(&evt) {
							count++
						}

						return nil
					})
					if err != nil {
						log.E.F("badger: count value read error: %s\n", err)
					}
				}
			}
		}

		return nil
	})

	return count, err
}

func (b *BadgerBackend) DeleteEvent(ctx context.Context,
	evt *nip1.Event) error {
	deletionHappened := false

	err := b.Update(func(txn *badger.Txn) error {
		idx := make([]byte, 1, 5)
		idx[0] = rawEventStorePrefix

		// query event by id to get its idx
		idPrefix8, _ := hex.DecodeString(string(evt.ID)[0 : 8*2])
		prefix := make([]byte, 1+8)
		prefix[0] = indexIdPrefix
		copy(prefix[1:], idPrefix8)
		opts := badger.IteratorOptions{
			PrefetchValues: false,
		}
		it := txn.NewIterator(opts)
		it.Seek(prefix)
		if it.ValidForPrefix(prefix) {
			idx = append(idx, it.Item().Key()[1+8:]...)
		}
		it.Close()

		// if no idx was found, end here, this event doesn't exist
		if len(idx) == 1 {
			return nil
		}

		// set this so we'll run the GC later
		deletionHappened = true

		// calculate all index keys we have for this event and delete them
		for _, k := range getIndexKeysForEvent(evt, idx[1:]) {
			if err := txn.Delete(k); err != nil {
				return err
			}
		}

		// delete the raw event
		return txn.Delete(idx)
	})
	if err != nil {
		return err
	}

	// after deleting, run garbage collector (sometimes)
	if deletionHappened {
		serialDelete = (serialDelete + 1) % 256
		if serialDelete == 0 {
			if err = b.RunValueLogGC(0.8); err != nil && !errors.Is(err,
				badger.ErrNoRewrite) {
				log.E.Ln("badger gc errored:" + err.Error())
			}
		}
	}

	return nil
}
