package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/sizer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
)

func (b *Backend) SaveEvent(c context.T, ev *event.T) (err error) {
	return b.Update(func(txn *badger.Txn) error {
		// make sure Close waits for this to complete
		b.WG.Add(1)
		defer b.WG.Done()
		// query event by id to ensure we don't save duplicates
		prefix := index.Id.Key(id.New(ev.ID))
		it := txn.NewIterator(badger.IteratorOptions{})
		defer it.Close()
		it.Seek(prefix)
		if it.ValidForPrefix(prefix) {
			// event exists
			return eventstore.ErrDupEvent
		}
		// encode to binary
		var bin []byte
		if bin, err = nostrbinary.Marshal(ev); chk.D(err) {
			return err
		}
		idx, ser := b.SerialKey()
		// raw event store
		if err = txn.Set(idx, bin); chk.D(err) {
			return err
		}
		var keyz [][]byte
		keyz = GetIndexKeysForEvent(ev, ser)
		for _, k := range keyz {
			if err = txn.Set(k, nil); chk.D(err) {
				return err
			}
		}
		// initialise access counter key
		counterKey := GetCounterKey(&ev.ID, ser)
		val := keys.Write(createdat.New(timestamp.Now()), sizer.New(len(bin)))
		log.T.F("counter %x %x", counterKey, val)
		if err = txn.Set(counterKey, val); chk.D(err) {
			return err
		}
		log.T.F("event saved")
		// if L2 is configured, write event out to it
		if b.L2 != nil {

		}
		return nil
	})
}
