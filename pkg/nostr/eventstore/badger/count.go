package badger

import (
	"encoding/binary"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"github.com/dgraph-io/badger/v4"
)

func (b *Backend) CountEvents(c context.T, f *filter.T) (count int, err error) {
	var queries []query
	var extraFilter *filter.T
	var since uint64
	if queries, extraFilter, since, err = PrepareQueries(f); chk.E(err) {
		return
	}
	var found [][]byte
	for _, q := range queries {
		// log.I.Ln("running count query", i)
		select {
		case <-c.Done():
			log.I.Ln("websocket closed")
			return
		case <-b.Ctx.Done():
			log.I.Ln("backend context canceled")
			return
		default:
		}
		var counted bool
		go func(q query) {
			err := b.View(func(txn *badger.Txn) (err error) {
				// iterate only through keys and in reverse order
				opts := badger.IteratorOptions{
					Reverse: true,
				}
				it := txn.NewIterator(opts)
				defer it.Close()
				for it.Seek(q.start); it.ValidForPrefix(q.searchPrefix); it.Next() {
					select {
					case <-c.Done():
						log.I.Ln("websocket closed")
						return
					case <-b.Ctx.Done():
						log.I.Ln("backend context canceled")
						return
					default:
					}
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
					if extraFilter == nil {
						count++
						counted = true
						return
					}
					ser := serial.FromKey(key)
					found = append(found, index.Event.Key(ser))
				}
				return
			})
			chk.E(err)
			log.I.Ln("closing results chan")
			close(q.results)
			for res := range q.results {
				log.I.Ln("closing results chan", res.Ev.ToObject().String())
			}
			log.T.Ln("count results channel clear",
				text.Trunc(q.queryFilter.ToObject().String()))
		}(q)
		if counted {
			continue
		}
		// if there was an extra filter
		for i := range found {
			evt := &event.T{}
			val := make([]byte, b.MaxLimit)
			err = b.View(func(txn *badger.Txn) (err error) {
				// iterate only through keys and in reverse order
				opts := badger.IteratorOptions{
					Reverse: true,
				}
				it := txn.NewIterator(opts)
				defer it.Close()
				for it.Rewind(); it.ValidForPrefix(found[i]); it.Next() {
					val, err = it.Item().ValueCopy(nil)
					if evt, err = nostrbinary.Unmarshal(val); chk.E(err) {
						return err
					}
					// check if this matches the other filters that were not
					// part of the index
					if extraFilter == nil || extraFilter.Matches(evt) {
						count++
					}
					return
				}
				return
			})
			chk.E(err)
		}
	}
	return count, err
}
