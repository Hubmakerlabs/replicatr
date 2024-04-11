package badger

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/dgraph-io/badger/v4"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/eventstore/badger/keys/createdat"
	"mleku.dev/git/nostr/eventstore/badger/keys/index"
	"mleku.dev/git/nostr/eventstore/badger/keys/serial"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/nostrbinary"
)

func (b *Backend) CountEvents(c context.T, f *filter.T) (count int, err error) {
	var queries []query
	var extraFilter *filter.T
	var since uint64
	queries, extraFilter, since, err = prepareQueries(f)
	if err != nil {
		return 0, err
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
				log.T.Ln("adding access to", acc.EvID)
				accesses = append(accesses, AccessEvent{acc.EvID, acc.Ser})
			}
		}
	}()
	err = b.View(func(txn *badger.Txn) (err error) {
		// iterate only through keys and in reverse order
		opts := badger.IteratorOptions{
			Reverse: true,
		}
		// actually iterate
		for _, q := range queries {
			select {
			case <-c.Done():
				err = log.W.Err("shutting down")
				return
			default:
			}
			txMx.Lock()
			it := txn.NewIterator(opts)
			txMx.Unlock()
			defer it.Close()
			for it.Seek(q.start); it.ValidForPrefix(q.prefix); it.Next() {
				item := it.Item()
				key := item.Key()

				// this is where the idx actually starts
				idxOffset := len(key) - SerialLen
				// "id" indexes don't contain a timestamp
				if !q.skipTS {
					createdAt := binary.BigEndian.Uint64(
						key[idxOffset-createdat.Len : idxOffset])
					if createdAt < since {
						break
					}
				}
				// ser := serial.New(nil)
				// keys.Read(key, prefix.New(prefixes.Event), ser)
				// idx := make([]byte, 1+SerialLen)
				// idx[0] = prefixes.Event.Byte()
				// copy(idx[1:], key[idxOffset:])
				ser := key[len(key)-serial.Len:]
				idx := index.Event.Key(serial.New(ser))
				if extraFilter == nil {
					count++
				} else {
					// fetch actual event
					item, err = txn.Get(idx)
					if err != nil {
						if errors.Is(err, badger.ErrDiscardedTxn) {
							return
						}
						log.D.F("badger: count (%v) failed to get %d "+
							"from raw event store: %s", q, idx)
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
							accessChan <- AccessEvent{EvID: &evt.ID, Ser: ser}
						}
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
