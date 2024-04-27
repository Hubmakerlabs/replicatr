package badger

import (
	"container/heap"
	"encoding/binary"
	"errors"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/priority"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
	"github.com/minio/sha256-simd"
)

func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C, err error) {
	ch = make(event.C)

	var queries []query
	var extraFilter *filter.T
	var since uint64
	queries, extraFilter, since, err = PrepareQueries(f)
	if chk.E(err) {
		return
	}
	// log.T.S(queries, extraFilter, since)
	accessChan := make(chan *AccessEvent)
	var txMx sync.Mutex
	// start up the access counter
	go b.AccessLoop(c, &txMx, accessChan)
	go func() {
		defer close(ch)
		// actually iterate
		for _, q1 := range queries {
			select {
			case <-c.Done():
				// log.I.Ln("websocket closed")
				return
			case <-b.Ctx.Done():
				// log.I.Ln("backend context canceled")
				return
			default:
			}
			q2 := q1
			go func() {
				err := b.View(func(txn *badger.Txn) (err error) {
					// iterate only through keys and in reverse order
					opts := badger.IteratorOptions{
						Reverse: true,
					}
					txMx.Lock()
					it := txn.NewIterator(opts)
					txMx.Unlock()
					defer it.Close()
					defer close(q2.results)
					for it.Seek(q2.start); it.ValidForPrefix(q2.searchPrefix); it.Next() {
						item := it.Item()
						key := item.Key()
						log.T.F("key %d %0x", len(key), key)
						idxOffset := len(key) - serial.Len
						// "id" indexes don't contain a timestamp
						if !q2.skipTS {
							createdAt := binary.BigEndian.Uint64(key[idxOffset-serial.Len : idxOffset])
							if createdAt < since {
								break
							}
						}
						log.T.F("found key %0x", key)
						ser := key[len(key)-serial.Len:]
						log.T.F("ser %0x", ser)
						idx := index.Event.Key(serial.New(ser))
						log.T.F("idx %0x", idx)
						// fetch actual event
						item, err = txn.Get(idx)
						if chk.T(err) {
							if errors.Is(err, badger.ErrDiscardedTxn) {
								return err
							}
							log.T.F("badger: failed to get %x based on prefix %x, "+
								"index key %x from raw event store: %s\n",
								idx, q2.searchPrefix, key, err)
							return err
						}
						err = item.Value(func(val []byte) (err error) {
							evt := &event.T{}
							if len(val) == sha256.Size {
								// this is a stub entry that indicates an L2 needs to be accessed for it, so we
								// populate only the event.T.ID and return the result.
								//
								// We can ignore the error because we know for certain by the test above it is
								// the right length.
								log.T.F("found event stub %0x must seek in L2", val)
								evt.ID, _ = eventid.New(hex.Enc(val))
								q2.results <- Results{Ev: evt, TS: timestamp.Now(), Ser: string(ser)}
								return
							}
							if evt, err = nostrbinary.Unmarshal(val); chk.E(err) {
								return
							}
							if evt == nil {
								log.D.S("got nil event from", val)
							}
							// check if this matches the other filters that were not part of the index
							if extraFilter == nil || extraFilter.Matches(evt) {
								res := Results{Ev: evt, TS: timestamp.Now(), Ser: string(ser)}
								log.T.F("ser result %s %d %0x", res.Ev.ID, res.TS, []byte(res.Ser))
								q2.results <- res
							}
							return
						})
						chk.E(err)
					}
					return nil
				})
				chk.T(err)
			}()
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
				log.T.F("adding event to queue %d %0x %0x", evt.Ev.ID, evt.TS.U64(), []byte(evt.Ser))
				emitQueue = append(emitQueue,
					&priority.QueryEvent{
						T:     evt.Ev,
						Query: q.index,
						Ser:   []byte(evt.Ser),
					})
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
			// send ID to be incremented for access
			ae := MakeAccessEvent(latest.T.ID, string(latest.Ser))
			accessChan <- ae
			log.T.Ln("sent access event", ae)
			ch <- latest.T
			log.T.Ln("sent result event", ae)
			// stop when reaching limit
			emittedEvents++
			if emittedEvents == limit {
				log.D.Ln("emitted the limit amount of events", limit)
				break
			}
			// fetch a new one from query results and replace the previous one with it
			if evt, ok := <-queries[latest.Query].results; ok {
				log.T.Ln("adding event to queue", evt.TS.U64())
				emitQueue[0].T = evt.Ev
				emitQueue[0].Ser = []byte(evt.Ser)
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
		if chk.E(err) {
			log.D.F("badger: query txn error: %s", err)
		}
		log.T.Ln("completed query")
	}()

	return ch, nil
}
