package badger

import (
	"container/heap"
	"encoding/binary"
	"errors"
	"math"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/priority"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"github.com/dgraph-io/badger/v4"
	"github.com/minio/sha256-simd"
)

func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C, err error) {
	ch = make(event.C)

	var queries []query
	var extraFilter *filter.T
	since := uint64(math.MaxInt64)
	queries, extraFilter, since, err = PrepareQueries(f)
	if chk.E(err) {
		return
	}
	for i := range queries {
		log.W.F("query %d %s", i, queries[i].queryFilter.ToObject().String())
	}
	// log.T.S(queries, extraFilter, since)
	accessChan := make(chan *AccessEvent)
	var txMx sync.Mutex
	// start up the access counter
	go b.AccessLoop(c, &txMx, accessChan)
	// max number of events we'll return
	limit := b.MaxLimit
	if f.Limit != nil && *f.Limit > 0 && *f.Limit < limit {
		limit = *f.Limit
	}
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
				log.I.Ln("query", text.Trunc(q2.queryFilter.ToObject().String()))
				var eventKeys [][]byte
				err := b.View(func(txn *badger.Txn) (err error) {
					// iterate only through keys and in reverse order
					opts := badger.IteratorOptions{Reverse: true}
					it := txn.NewIterator(opts)
					defer it.Close()
					for it.Seek(q2.start); it.ValidForPrefix(q2.searchPrefix); it.Next() {
						item := it.Item()
						k := item.KeyCopy(nil)
						// log.I.F("item key %0x", k)
						if !q2.skipTS {
							createdAt := createdat.FromKey(k)
							// log.I.F("%d > %d", createdAt.Val.U64(), since)
							if createdAt.Val.U64() < since {
								// log.I.Ln("break")
								break
							}
						}
						ser := serial.FromKey(k)
						eventKeys = append(eventKeys, index.Event.Key(ser))
					}
					return
				})
				for _, eventKey := range eventKeys {
					log.I.F("event key %0x", eventKey)
					err = b.View(func(txn *badger.Txn) (err error) {
						opts := badger.IteratorOptions{Reverse: true}
						it := txn.NewIterator(opts)
						defer it.Close()
						for it.Seek(eventKey); it.ValidForPrefix(eventKey); it.Next() {
							item := it.Item()
							var v []byte
							if v, err = item.ValueCopy(nil); chk.E(err) {
								continue
							}
							if len(v) == sha256.Size {

							}
							// log.I.F("event binary %0x", v)
							var evt *event.T
							if evt, err = nostrbinary.Unmarshal(v); chk.E(err) {
								continue
							}
							log.W.F("key %0x val %s", item.KeyCopy(nil), evt.ToObject().String())
						}
						return
					})
				}
				if len(eventKeys) < 1 {
					log.W.Ln("nothing found for", text.Trunc(q2.queryFilter.ToObject().String()))
					// return
				}
				err = b.View(func(txn *badger.Txn) (err error) {
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
						// log.T.F("key %d %0x", len(key), key)
						idxOffset := len(key) - serial.Len
						// "id" indexes don't contain a timestamp
						if !q2.skipTS {
							createdAt := binary.BigEndian.Uint64(key[idxOffset-serial.Len : idxOffset])
							if createdAt < since {
								break
							}
						}
						ser := serial.FromKey(key)
						log.T.F("ser %d", ser.Uint64())
						idx := index.Event.Key(ser)
						log.T.F("idx %0x", idx)
						// fetch actual event
						item, err = txn.Get(idx)
						if err != nil {
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
								q2.results <- Results{Ev: evt, TS: timestamp.Now(), Ser: ser}
								return
							}
							if evt, err = nostrbinary.Unmarshal(val); chk.E(err) {
								return
							}
							if evt == nil {
								log.D.S("got nil event from", val)
								return
							}
							// check if this matches the other filters that were not part of the index
							if extraFilter == nil || extraFilter.Matches(evt) {
								res := Results{Ev: evt, TS: timestamp.Now(), Ser: ser}
								log.T.F("ser result %s %d %d", res.Ev.ID, res.TS, res.Ser.Uint64())
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
		// receive results and ensure we only return the most recent ones always
		emittedEvents := 0
		// first pass
		emitQueue := make(priority.Queue, 0, len(queries)+limit)
		for _, q := range queries {
			q := q
			evt, ok := <-q.results
			if ok {
				// log.T.F("adding event to queue [%s, %d]", evt.Ev.ID, evt.Ser.Uint64())
				emitQueue = append(emitQueue,
					&priority.QueryEvent{
						T:     evt.Ev,
						Query: q.index,
						Ser:   evt.Ser,
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
			ae := MakeAccessEvent(latest.T.ID, latest.Ser)
			// log.T.Ln("sending access event", ae)
			accessChan <- ae
			ch <- latest.T
			// stop when reaching limit
			emittedEvents++
			if emittedEvents == limit {
				// log.D.Ln("emitted the limit amount of events", limit)
				break
			}
			// fetch a new one from query results and replace the previous one with it
			if evt, ok := <-queries[latest.Query].results; ok {
				// log.T.Ln("adding event to queue", evt.TS.U64())
				emitQueue[0].T = evt.Ev
				emitQueue[0].Ser = evt.Ser
				heap.Fix(&emitQueue, 0)
			} else {
				// log.T.Ln("removing event from queue")
				// if this query has no more events we just remove this and proceed normally
				heap.Remove(&emitQueue, 0)
				// check if the list is empty and end
				if len(emitQueue) == 0 {
					// log.T.Ln("emit queue empty")
					break
				}
			}
		}
		if chk.E(err) {
			log.D.F("badger: query txn error: %s", err)
		}
		// log.T.Ln("completed query")
	}()

	return ch, nil
}
