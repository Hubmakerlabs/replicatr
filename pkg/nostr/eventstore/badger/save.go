package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"

	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/dgraph-io/badger/v4"
)

func (b *BadgerBackend) SaveEvent(c context.T, evt *event.T) (err error) {
	return b.Update(func(txn *badger.Txn) (err error) {
		b.D.Ln("saving event")
		// query event by id to ensure we don't save duplicates
		id, _ := hex.Dec(evt.ID.String())
		prefix := make([]byte, 1+8)
		prefix[0] = indexIdPrefix
		copy(prefix[1:], id)
		it := txn.NewIterator(badger.IteratorOptions{})
		defer it.Close()
		it.Seek(prefix)
		if it.ValidForPrefix(prefix) {
			// event exists
			return eventstore.ErrDupEvent
		}
		b.D.Ln("encoding to binary")
		// encode to binary
		var bin []byte
		if bin, err = nostrbinary.Marshal(evt); b.Fail(err) {
			return err
		}
		b.T.F("binary encoded %x", bin)
		idx := b.Serial()
		// raw event store
		b.D.F("setting event")
		if err = txn.Set(idx, bin); b.Fail(err) {
			return err
		}
		b.D.F("get index keys for event")
		for _, k := range getIndexKeysForEvent(evt, idx[1:]) {
			b.D.F("index key %x", k)
			if err = txn.Set(k, nil); b.Fail(err) {
				return err
			}
		}
		b.D.F("event saved")
		return nil
	})
}
