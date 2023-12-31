package badger

import (
	"context"
	"encoding/hex"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/dgraph-io/badger/v4"
)

var serialDelete uint32 = 0

func (b *BadgerBackend) DeleteEvent(ctx context.Context, evt *event.T) (e error) {
	deletionHappened := false

	e = b.Update(func(txn *badger.Txn) (e error) {
		idx := make([]byte, 1, 5)
		idx[0] = rawEventStorePrefix

		// query event by id to get its idx
		idPrefix8, _ := hex.DecodeString(evt.ID[0 : 8*2])
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
			if e := txn.Delete(k); e != nil {
				return e
			}
		}

		// delete the raw event
		return txn.Delete(idx)
	})
	if e != nil {
		return e
	}

	// after deleting, run garbage collector (sometimes)
	if deletionHappened {
		serialDelete = (serialDelete + 1) % 256
		if serialDelete == 0 {
			if e := b.RunValueLogGC(0.8); e != nil && e != badger.ErrNoRewrite {
				log.D.Ln("badger gc error:" + e.Error())
			}
		}
	}

	return nil
}
