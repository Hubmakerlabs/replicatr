package badger

import (
	"container/heap"
	"encoding/binary"
	"errors"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/priority"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/dgraph-io/badger/v4"
)

type query struct {
	i       int
	f       *filter.T
	prefix  []byte
	start   []byte
	results chan Results
	skipTS  bool
}

type Results struct {
	Ev  *event.T
	Ser []byte
}

func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C, err error) {
	ch = make(event.C)

	if f.Kinds != nil && len(f.Kinds) > 0 {
		for i := range f.Kinds {
			switch f.Kinds[i] {
			case kind.EncryptedDirectMessage,
				kind.GiftWrap,
				kind.GiftWrapWithKind4,
				kind.ApplicationSpecificData,
				kind.Deletion:
				log.T.Ln("privileged event kind in filter", f.String())
			}
		}
	}

	var queries []query
	var extraFilter *filter.T
	var since uint64
	queries, extraFilter, since, err = prepareQueries(f)
	if err != nil {
		return nil, err
	}
	accessChan := make(chan AccessEvent)
	var txMx sync.Mutex
	// start up the access counter
	go func() {
		var accesses []AccessEvent
		b.WG.Add(1)
		defer b.WG.Done()
		for {
			select {
			case <-c.Done():
				if len(accesses) > 0 {
					log.D.Ln("accesses", accesses)
					chk.E(b.IncrementAccesses(&txMx, accesses))
				}
				return
			case acc := <-accessChan:
				log.T.F("adding access to %s %0x",
					acc.EvID, acc.Ser)
				accesses = append(accesses, AccessEvent{acc.EvID, acc.Ser})
			}
		}
	}()

	go func() {
		defer close(ch)
		// actually iterate
		for _, q := range queries {
			select {
			case <-c.Done():
				log.I.Ln("websocket closed")
				return
			case <-b.Ctx.Done():
				log.I.Ln("backend context canceled")
				return
			default:
			}
			q := q
			go b.View(func(txn *badger.Txn) error {
				// iterate only through keys and in reverse order
				opts := badger.IteratorOptions{
					Reverse: true,
				}
				txMx.Lock()
				it := txn.NewIterator(opts)
				txMx.Unlock()
				defer it.Close()
				defer close(q.results)
				for it.Seek(q.start); it.ValidForPrefix(q.prefix); it.Next() {
					item := it.Item()
					key := item.Key()
					idxOffset := len(key) - serial.Len
					// idxOffset := len(key) - 8 // this is where the idx actually starts
					// "id" indexes don't contain a timestamp
					if !q.skipTS {
						createdAt := binary.BigEndian.Uint64(key[idxOffset-serial.Len : idxOffset])
						if createdAt < since {
							break
						}
					}
					// idx := make([]byte, 5)
					// idx[0] = rawEventStorePrefix
					log.I.F("found key %0x", key)
					ser := key[len(key)-serial.Len:]
					log.I.F("ser %0x", ser)
					idx := index.Event.Key(serial.New(ser))
					log.I.F("idx %0x", idx)
					// log.I.F("ser %0x idx %0x", ser, idx)
					// copy(idx[1:], key[idxOffset:])
					// fetch actual event
					item, err = txn.Get(idx)
					if err != nil {
						if errors.Is(err, badger.ErrDiscardedTxn) {
							return err
						}
						log.E.F("badger: failed to get %x based on prefix %x, "+
							"index key %x from raw event store: %s\n",
							idx, q.prefix, key, err)
						return err
					}
					item.Value(func(val []byte) error {
						evt := &event.T{}
						if evt, err = nostrbinary.Unmarshal(val); err != nil {
							log.E.F("badger: value read error (id %x): %s\n", val[0:32], err)
							return err
						}
						if evt == nil {
							log.D.S("got nil event from", val)
						}
						// check if this matches the other filters that were not part of the index
						if extraFilter == nil || extraFilter.Matches(evt) {
							log.I.F("ser result %0x", ser)
							q.results <- Results{Ev: evt, Ser: ser}
							// accessChan <- AccessEvent{EvID: &evt.ID, Ser: ser}
							// q.results <- evt
						}
						return nil
					})
				}
				return nil
			})
		}
		// max number of events we'll return
		limit := b.MaxLimit
		if f.Limit != nil && *f.Limit > 0 && *f.Limit < limit {
			limit = *f.Limit
		}

		// receive results and ensure we only return the most recent ones always
		emittedEvents := 0

		// first pass
		emitQueue := make(priority.Queue, 0, len(queries)+limit)
		for _, q := range queries {
			evt, ok := <-q.results
			if ok {
				emitQueue = append(emitQueue, &priority.QueryEvent{T: evt.Ev, Query: q.i, Ser: evt.Ser})
			}
		}
		// queue may be empty here if we have literally nothing
		if len(emitQueue) == 0 {
			return
		}
		heap.Init(&emitQueue)
		// iterate until we've emitted all events required
		for {
			select {
			case <-c.Done():
				// websocket closed
				log.T.Ln("websocket closed")
				return
			case <-b.Ctx.Done():
				// backend context canceled
				log.T.Ln("backend context canceled")
				return
			default:
			}
			// emit latest event in queue
			latest := emitQueue[0]
			ch <- latest.T
			// send ID to be incremented for access
			accessChan <- AccessEvent{EvID: &latest.T.ID, Ser: latest.Ser}
			// stop when reaching limit
			emittedEvents++
			if emittedEvents == limit {
				log.D.Ln("emitted the limit amount of events", limit)
				break
			}
			// fetch a new one from query results and replace the previous one with it
			if evt, ok := <-queries[latest.Query].results; ok {
				log.T.Ln("adding event to queue")
				emitQueue[0].T = evt.Ev
				heap.Fix(&emitQueue, 0)
			} else {
				log.T.Ln("removing event from queue")
				// if this query has no more events we just remove this and proceed normally
				heap.Remove(&emitQueue, 0)
				// check if the list is empty and end
				if len(emitQueue) == 0 {
					log.T.Ln("emit queue empty")
					break
				}
			}
		}
		if err != nil {
			log.D.F("badger: query txn error: %s", err)
		}
		log.T.Ln("completed query")
	}()

	return ch, nil
}
