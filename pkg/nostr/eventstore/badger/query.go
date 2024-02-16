package badger

import (
	"container/heap"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/dgraph-io/badger/v4"
)

type query struct {
	i             int
	f             *filter.T
	prefix        []byte
	startingPoint []byte // todo: rework this to use 64 bit numbers
	results       chan *event.T
	skipTimestamp bool
}

type queryEvent struct {
	*event.T
	query int
}

func (b *Backend) QueryEvents(c context.T, f *filter.T) (chan *event.T, error) {
	ch := make(chan *event.T)

	// log.D.Ln("preparing queries")
	queries, extraFilter, since, err := prepareQueries(f)
	if err != nil {
		return nil, err
	}
	// log.D.Ln("scanning database")
	go func() {
		err := b.View(func(txn *badger.Txn) (err error) {
			// iterate only through keys and in reverse order
			opts := badger.IteratorOptions{
				Reverse: true,
			}
			var txMx sync.Mutex
			// actually iterate
			iteratorClosers := make([]func(), len(queries))
			for i, q := range queries {
				// txMx.Lock()
				// go func(i int, q query) {
				// var err error
				txMx.Lock()
				it := txn.NewIterator(opts)
				txMx.Unlock()
				iteratorClosers[i] = it.Close

				if q.startingPoint == nil {
					log.D.S("nil query starting point")
					return
				}
				if q.prefix == nil {
					log.D.S("nil query prefix")
					return
				}
				for it.Seek(q.startingPoint); it.ValidForPrefix(q.prefix); it.Next() {
					item := it.Item()
					// log.D.Ln(item.ValueCopy(nil))
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

					// fetch actual event
					item, err = txn.Get(idx)
					if err != nil {
						if errors.Is(err, badger.ErrDiscardedTxn) {
							return
						}
						log.D.F("badger: failed to get %x based on prefix %x, index key %x from raw event store: %s",
							idx, q.prefix, key, err)
						return
					}
					if item == nil {
						log.D.Ln("nil item")
						break
					}
					var val []byte
					val, err = item.ValueCopy(val)
					evt := &event.T{}
					if evt, err = nostrbinary.Unmarshal(val); err != nil {
						log.D.S("badger: value read error", val, err)
						break
					}
					if evt == nil {
						log.D.S("got nil event from", val)
					}
					// log.D.Ln("unmarshaled", evt.ToObject().String())
					// check if this matches the other filters that were not part of the index
					if extraFilter == nil || extraFilter.Matches(evt) {
						// log.D.Ln("dispatching event to results queue", evt == nil)
						q.results <- evt
						// log.D.Ln("results queue was consumed")
					}
				}
				// }(i, q)
				// txMx.Unlock()
			}

			// max number of events we'll return
			limit := b.MaxLimit
			if f.Limit != nil {
				if *f.Limit > 0 && *f.Limit < limit {
					limit = *f.Limit
				}
			}

			// // default an empty or zero limit to 1 so replaceable events come
			// // out as expected.
			// var isReplaceable bool
			// for i := range f.Kinds {
			// 	if f.Kinds[i].IsReplaceable() || f.Kinds[i].IsParameterizedReplaceable() {
			// 		isReplaceable = true
			// 		break
			// 	}
			// }
			// // note that this will trigger if replaceable event kinds appear in
			// // a filter, even if not all are replaceable, so if this happens and
			// // the limit is zero then it will be set to 1 result only, which may
			// // not be what the querant intends, however, they should specify a
			// // limit for replaceable events as they should only want the newest
			// // one
			// //
			// // this ensures that clients that are looking for replaceable events
			// // and don't specify a limit they get the expected single, newest
			// // version
			// if isReplaceable {
			// 	if f.Limit == nil {
			// 		log.D.Ln("setting limit to 1 for missing limit to handle replaceable events correctly")
			// 		limit = 1
			// 	}
			// } else {
			// 	if f.Limit != nil && *f.Limit == 0 {
			// 		log.D.Ln("setting limit to max for zero limit to handle non-replaceable events correctly:", b.MaxLimit)
			// 		limit = b.MaxLimit
			// 	}
			// 	if len(f.Kinds) == 1 && f.Kinds[0].IsEphemeral() {
			// 		log.D.Ln("setting limit to 0 for ephemeral event, only one to be returned")
			// 		limit = 0
			// 	}
			// }

			// receive results and ensure we only return the most recent ones always
			emittedEvents := 0

			// first pass
			emitQueue := NewPriorityQueue(len(queries) + limit)
			for i, q := range queries {
				// log.T.Ln("receiving query", i, q.f.ToObject().String())
				select {
				case evt := <-q.results:
					log.T.F("returning event from query %d %s\nEVENT: %s", i,
						q.f.ToObject().String(), evt.ToObject().String())
					emitQueue.Queries = append(emitQueue.Queries, &queryEvent{T: evt, query: q.i})
				}
			}

			// now it's a good time to schedule this
			defer func() {
				log.D.Ln("closing iterators")
				txMx.Lock()
				defer txMx.Unlock()
				for _, itclose := range iteratorClosers {
					itclose()
				}
				close(ch)
				for _, q := range queries {
					close(q.results)
				}
			}()

			// queue may be empty here if we have literally nothing
			if emitQueue.Len() == 0 {
				log.D.Ln("emit queue is empty")
				return nil
			}

			log.D.Ln("initialising emit queue")
			heap.Init(emitQueue)

			// iterate until we've emitted all events required
			for {
				// emit latest event in queue
				log.D.Ln("emitting latest event in queue")
				latest := emitQueue.Queries[0]
				ch <- latest.T
				log.D.Ln("emitted event")
				// stop when reaching limit
				emittedEvents++
				if emittedEvents == limit {
					log.D.Ln("emitted the limit amount of events", limit)
					break
				}

				// fetch a new one from query results and replace the previous
				// one with it
				if evt, ok := <-queries[latest.query].results; ok {
					log.D.Ln("adding event to queue")
					emitQueue.Queries[0].T = evt
					heap.Fix(emitQueue, 0)
				} else {
					log.D.Ln("removing event from queue")
					// if this query has no more events we just remove this and proceed normally
					heap.Remove(emitQueue, 0)

					// check if the list is empty and end
					if emitQueue.Len() == 0 {
						log.D.Ln("emit queue empty")
						break
					}
				}
			}
			return nil
		})

		if err != nil {
			log.D.F("badger: query txn error: %s", err)
		}
	}()
	// log.D.Ln("completed query")
	return ch, nil
}

func prepareQueries(f *filter.T) (
	queries []query,
	extraFilter *filter.T,
	since uint32,
	err error,
) {
	var index byte

	if len(f.IDs) > 0 {
		index = indexIdPrefix
		queries = make([]query, len(f.IDs))
		for i, idHex := range f.IDs {
			prefix := make([]byte, 1+8)
			prefix[0] = index
			if len(idHex) != 64 {
				return nil, nil, 0, fmt.Errorf("invalid id '%s'", idHex)
			}
			idPrefix8, _ := hex.Dec(idHex[0 : 8*2])
			copy(prefix[1:], idPrefix8)
			queries[i] = query{i: i, f: f, prefix: prefix, skipTimestamp: true}
		}
	} else if len(f.Authors) > 0 {
		if len(f.Kinds) == 0 {
			index = indexPubkeyPrefix
			queries = make([]query, len(f.Authors))
			for i, pubkeyHex := range f.Authors {
				if len(pubkeyHex) != 64 {
					// todo: some clients are sending invalid pubkeyhex of 69 chars
					return nil, nil, 0, fmt.Errorf("invalid pubkey '%s'", pubkeyHex)
				}
				pubkeyPrefix8, _ := hex.Dec(pubkeyHex[0 : 8*2])
				prefix := make([]byte, 1+8)
				prefix[0] = index
				copy(prefix[1:], pubkeyPrefix8)
				queries[i] = query{i: i, f: f, prefix: prefix}
			}
		} else {
			index = indexPubkeyKindPrefix
			queries = make([]query, len(f.Authors)*len(f.Kinds))
			i := 0
			for _, pubkeyHex := range f.Authors {
				for _, kind := range f.Kinds {
					if len(pubkeyHex) != 64 {
						// todo: some clients are sending invalid pubkeyhex of 69 chars
						// eg; ["NOTICE","invalid pubkey '0020bf2376e17ba4ec269d10fcc996a4746b451152be9031fa48e74553dde5526bce'"]
						return nil, nil, 0, fmt.Errorf("invalid pubkey '%s'", pubkeyHex)
					}
					pubkeyPrefix8, _ := hex.Dec(pubkeyHex[0 : 8*2])
					prefix := make([]byte, 1+8+2)
					prefix[0] = index
					copy(prefix[1:], pubkeyPrefix8)
					binary.BigEndian.PutUint16(prefix[1+8:], uint16(kind))
					queries[i] = query{i: i, f: f, prefix: prefix}
					i++
				}
			}
		}
		extraFilter = &filter.T{Tags: f.Tags}
	} else if len(f.Tags) > 0 {
		// determine the size of the queries array by inspecting all tags sizes
		size := 0
		for _, values := range f.Tags {
			size += len(values)
		}
		if size == 0 {
			return nil, nil, 0, fmt.Errorf("empty tag filters")
		}

		queries = make([]query, size)

		extraFilter = &filter.T{Kinds: f.Kinds}
		i := 0
		for _, values := range f.Tags {
			for _, value := range values {
				// get key prefix (with full length) and offset where to write the last parts
				k, offset := getTagIndexPrefix(value)
				// remove the last parts part to get just the prefix we want here
				prefix := k[0:offset]

				queries[i] = query{i: i, f: f, prefix: prefix}
				i++
			}
		}
	} else if len(f.Kinds) > 0 {
		index = indexKindPrefix
		queries = make([]query, len(f.Kinds))
		for i, kind := range f.Kinds {
			prefix := make([]byte, 1+2)
			prefix[0] = index
			binary.BigEndian.PutUint16(prefix[1:], uint16(kind))
			queries[i] = query{i: i, f: f, prefix: prefix}
		}
	} else {
		index = indexCreatedAtPrefix
		queries = make([]query, 1)
		prefix := make([]byte, 1)
		prefix[0] = index
		queries[0] = query{i: 0, f: f, prefix: prefix}
		extraFilter = nil
	}

	var until uint32 = math.MaxUint32
	if f.Until != nil {
		if fu := uint32(*f.Until); fu < until {
			until = fu + 1
		}
	}
	for i, q := range queries {
		queries[i].startingPoint = binary.BigEndian.AppendUint32(q.prefix, uint32(until))
		queries[i].results = make(chan *event.T, 12)
	}

	// this is where we'll end the iteration
	if f.Since != nil {
		if fs := uint32(*f.Since); fs > since {
			since = fs
		}
	}

	return queries, extraFilter, since, nil
}
