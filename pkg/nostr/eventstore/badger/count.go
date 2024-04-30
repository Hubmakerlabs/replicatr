package badger

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/dgraph-io/badger/v4"
)

func (b *Backend) CountEvents(c context.T, f *filter.T) (count int, err error) {
	var queries []query
	var extraFilter *filter.T
	var since uint64
	// log.I.Ln("preparing queries for count")
	if queries, extraFilter, since, err = PrepareQueries(f); chk.E(err) {
		return
	}
	accessChan := make(chan *AccessEvent)
	var txMx sync.Mutex
	// start up the access counter
	// log.I.Ln("starting access counter for count")
	go b.AccessLoop(c, &txMx, accessChan)
	err = b.View(func(txn *badger.Txn) (err error) {
		// log.I.Ln("running query")
		// iterate only through keys and in reverse order
		opts := badger.IteratorOptions{
			Reverse: true,
		}
		// actually iterate
		for _, q := range queries {
			// log.I.Ln("running count query", i)
			select {
			case <-c.Done():
				err = log.W.Err("shutting down")
				return
			default:
			}
			// log.I.Ln("creating new iterator", i)
			txMx.Lock()
			it := txn.NewIterator(opts)
			txMx.Unlock()
			// log.I.Ln("defer iterator to close", i)
			defer it.Close()
			for it.Seek(q.start); it.ValidForPrefix(q.searchPrefix); it.Next() {
				item := it.Item()
				key := item.Key()
				// this is where the idx actually starts
				idxOffset := len(key) - serial.Len
				// "id" indexes don't contain a timestamp
				if !q.skipTS {
					createdAt := binary.BigEndian.Uint64(
						key[idxOffset-createdat.Len : idxOffset])
					if createdAt < since {
						break
					}
				}
				ser := serial.FromKey(key)
				idx := index.Event.Key(ser)
				if extraFilter == nil {
					log.I.F("adding access for count (no extra filter) %d", ser.Uint64())
					count++
				} else {
					// log.I.Ln("fetching actual event", i)
					// fetch actual event
					if item, err = txn.Get(idx); chk.E(err) {
						if errors.Is(err, badger.ErrDiscardedTxn) {
							return
						}
						// log.D.F("badger: count (%v) failed to get %d "+
						// 	"from raw event store: %s", q, idx)
						return
					}
					err = item.Value(func(val []byte) (err error) {
						var evt *event.T
						if evt, err = nostrbinary.Unmarshal(
							val); chk.E(err) {
							return err
						}
						// check if this matches the other filters that were not
						// part of the index
						if extraFilter == nil || extraFilter.Matches(evt) {
							count++
						}
						// log.I.F("adding access for count %s %0x", evt.ID, ser)
						accessChan <- &AccessEvent{EvID: evt.ID, Ser: ser}
						return nil
					})
					if chk.D(err) {
						log.D.F("badger: count value read error: %s", err)
					}
				}
			}
		}
		return nil
	})
	return count, err
}
