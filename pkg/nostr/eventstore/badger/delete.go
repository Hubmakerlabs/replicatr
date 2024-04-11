package badger

import (
	"errors"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/dgraph-io/badger/v4"
)

var serialDelete uint32 = 0

func (b *Backend) DeleteEvent(c context.T, ev *event.T) error {
	deletionHappened := false

	err := b.Update(func(txn *badger.Txn) error {
		idx := make([]byte, 1, 1+serial.Len)
		// idx[0] = rawEventStorePrefix

		// query event by id to get its idx
		// idPrefix8, _ := hex.DecodeString(ev.ID[0 : 8*2].String())
		// prefix := make([]byte, 1+8)
		// prefix[0] = indexIdPrefix
		// copy(prefix[1:], idPrefix8)
		idKey := index.Id.Key(id.New(ev.ID))
		opts := badger.IteratorOptions{
			PrefetchValues: false,
		}
		it := txn.NewIterator(opts)
		it.Seek(idKey)
		if it.ValidForPrefix(idKey) {
			// we only need the serial to generate the event key
			ser := serial.New(nil)
			keys.Read(it.Item().Key(), index.Empty(), id.New(""), ser)
			// idx = append(idx, it.Item().Key()[1+8:]...)
			idx = index.Event.Key(ser)
			log.D.Ln("added found item")
		}
		it.Close()
		// if no idx was found, end here, this event doesn't exist
		if len(idx) == 1 {
			return nil
		}
		// set this so we'll run the GC later
		deletionHappened = true
		// calculate all index keys we have for this event and delete them
		for _, k := range GetIndexKeysForEvent(ev, idx[1:]) {
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
			if err := b.RunValueLogGC(0.8); err != nil &&
				!errors.Is(err, badger.ErrNoRewrite) {
				log.E.F("badger gc errored:" + err.Error())
			}
		}
	}
	return nil
}
