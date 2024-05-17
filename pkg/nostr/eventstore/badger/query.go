package badger

import (
	"container/heap"
	"math"

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
	"github.com/dgraph-io/badger/v4"
	"github.com/minio/sha256-simd"
)

func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C, err error) {
	ch = make(event.C)

	// log.I.Ln("QueryEvents", f.ToObject().String())
	var queries []query
	var extraFilter *filter.T
	since := uint64(math.MaxInt64)
	queries, extraFilter, since, err = PrepareQueries(f)
	if chk.E(err) {
		return
	}
	accessChan := make(chan *AccessEvent)
	// start up the access counter
	go b.AccessLoop(c, accessChan)
	// max number of events we'll return
	limit := b.MaxLimit
	if f.Limit != nil && *f.Limit > 0 && *f.Limit < limit {
		limit = *f.Limit
	}
	go func() {
		defer func() {
			log.T.Ln("closing query chan", f.ToObject().String(), b.Path)
			close(ch)
			log.T.Ln("closing access chan", f.ToObject().String(), b.Path)
			close(accessChan)
			// log.I.Ln("query goroutine terminated")
		}()
		// actually iterate
		for _, q1 := range queries {
			// log.I.Ln("query", q1.queryFilter.ToObject().String())
			select {
			case <-c.Done():
				log.T.Ln("websocket closed")
				return
			case <-b.Ctx.Done():
				log.I.Ln("backend context canceled")
				return
			default:
			}
			q2 := q1
			go func() {
				// log.I.Ln("scanning database for", q2.queryFilter.ToObject().String())
				var eventKeys [][]byte
				err := b.View(func(txn *badger.Txn) (err error) {
					// iterate only through keys and in reverse order
					opts := badger.IteratorOptions{Reverse: true}
					it := txn.NewIterator(opts)
					defer it.Close()
					for it.Seek(q2.start); it.ValidForPrefix(q2.searchPrefix); it.Next() {
						item := it.Item()
						k := item.KeyCopy(nil)
						if !q2.skipTS {
							createdAt := createdat.FromKey(k)
							if createdAt.Val.U64() < since {
								break
							}
						}
						ser := serial.FromKey(k)
						eventKeys = append(eventKeys, index.Event.Key(ser))
					}
					return
				})
				// log.I.Ln("done scanning database", q2.queryFilter.ToObject().String())
				if err != nil {
					close(q2.results)
					for _ = range q2.results {
					}
					// log.I.Ln("query results channel clear",
					// 	text.Trunc(q2.queryFilter.ToObject().String()))
					return
				}
				// log.I.Ln("got eventKeys", eventKeys)
				for _, eventKey := range eventKeys {
					var ev *event.T
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
							ser := serial.FromKey(item.KeyCopy(nil))
							if len(v) == sha256.Size {
								// this is a stub entry that indicates an L2 needs to be accessed for it, so we
								// populate only the event.T.ID and return the result.
								evt := &event.T{}
								log.T.F("found event stub %0x must seek in L2", v)
								evt.ID, _ = eventid.New(hex.Enc(v))
								select {
								case <-c.Done():
									// log.I.Ln("websocket closed")
									return
								case <-b.Ctx.Done():
									log.I.Ln("backend context canceled")
									return
								default:
								}
								q2.results <- Results{Ev: evt, TS: timestamp.Now(), Ser: ser}
								return
							}
							if ev, err = nostrbinary.Unmarshal(v); chk.E(err) {
								continue
							}
							if ev == nil {
								log.D.S("got nil event from", v)
								return
							}
							// check if this matches the other filters that were not part of the index
							if extraFilter == nil || extraFilter.Matches(ev) {
								res := Results{Ev: ev, TS: timestamp.Now(), Ser: ser}
								// log.W.F("key %d val %s", serial.FromKey(item.KeyCopy(nil)).Uint64(),
								// 	ev.ToObject().String())
								// select {
								// case <-c.Done():
								// 	// log.I.Ln("websocket closed")
								// 	return
								// case <-b.Ctx.Done():
								// 	log.I.Ln("backend context canceled")
								// 	return
								// default:
								// }
								q2.results <- res
							}
						}
						// close(q2.results)
						return
					})
				}
				// q2.results <- Results{}
				// log.I.Ln("closing results channel")
				close(q2.results)
				// log.I.Ln("draining results channel")
				// for _ = range q2.results {
				// }
				// log.I.Ln("results channel clear",
				// 	text.Trunc(q2.queryFilter.ToObject().String()))
			}()
		}
		// log.I.Ln("finished queries loop", queries)
		// receive results and ensure we only return the most recent ones always
		emittedEvents := 0
		// first pass
		emitQueue := make(priority.Queue, 0, len(queries)+limit)
		for _, q := range queries {
			// log.I.Ln(q.queryFilter.ToObject().String())
			q := q
			evt, ok := <-q.results
			log.I.Ln("received result", ok, evt)
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
		// log.I.Ln("len emitQueue")
		// queue may be empty here if we have literally nothing
		if len(emitQueue) == 0 {
			// log.I.Ln("emit queue empty")
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
			log.T.Ln("sending result", latest.T.ToObject().String(), b.Path)
			ch <- latest.T
			log.T.Ln("sent result", latest.T.ToObject().String(), b.Path)
			// send ID to be incremented for access
			ae := MakeAccessEvent(latest.T.ID, latest.Ser)
			log.T.Ln("sending access event", ae, b.Path)
			accessChan <- ae
			log.T.Ln("sent access event", ae, b.Path)
			// stop when reaching limit
			emittedEvents++
			if emittedEvents == limit {
				// log.D.Ln("emitted the limit amount of events", limit)
				break
			}
			// log.I.Ln("fetching new event from queue")
			// fetch a new one from query results and replace the previous one with it
			if evt, ok := <-queries[latest.Query].results; ok {
				emitQueue[0].T = evt.Ev
				emitQueue[0].Ser = evt.Ser
				heap.Fix(&emitQueue, 0)
			} else {
				// if this query has no more events we just remove this and proceed normally
				heap.Remove(&emitQueue, 0)
				// check if the list is empty and end
				if len(emitQueue) == 0 {
					// log.I.Ln("event results queue finished")
					break
				}
			}
		}
		if chk.E(err) {
			log.D.F("badger: query txn error: %s", err)
		}
		// log.I.Ln("query complete")
	}()
	// log.I.Ln("passing results channel back to caller")
	return ch, nil
}
