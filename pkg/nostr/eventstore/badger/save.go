package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"

	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/dgraph-io/badger/v4"
)

func (b *Backend) SaveEvent(c context.T, evt *event.T) (err error) {
	return b.Update(func(txn *badger.Txn) (err error) {
		// log.D.S(evt)
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
		log.T.Ln("encoding to binary")
		// encode to binary
		var bin []byte
		if bin, err = nostrbinary.Marshal(evt); chk.D(err) {
			return err
		}
		// log.D.F("binary encoded %x", bin)
		idx := b.Serial()
		// raw event store
		log.T.F("setting event")
		if err = txn.Set(idx, bin); chk.D(err) {
			return err
		}
		log.T.F("get index keys for event")
		for _, k := range getIndexKeysForEvent(evt, idx[1:]) {
			log.T.F("index key %x", k)
			if err = txn.Set(k, nil); chk.D(err) {
				return err
			}
		}
		log.T.F("event saved")
		return nil
	})
}
