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
		// id, _ := hex.Dec(ev.ID[:16].String())
		// prefix := make([]byte, 1+8)
		// prefix[0] =
		prefix := index.Id.Key(id.New(ev.ID))
		// copy(prefix[1:], id)
		it := txn.NewIterator(badger.IteratorOptions{})
		defer it.Close()
		it.Seek(prefix)
		if it.ValidForPrefix(prefix) {
			// event exists
			return eventstore.ErrDupEvent
		}
		// log.T.Ln("encoding to binary")
		// encode to binary
		var bin []byte
		if bin, err = nostrbinary.Marshal(ev); chk.D(err) {
			return err
		}
		idx, ser := b.SerialKey()
		log.W.F("idx %x", idx)
		// raw event store
		// log.T.F("setting event")
		if err = txn.Set(idx, bin); chk.D(err) {
			return err
		}
		// log.T.F("get index keys for event")
		var keyz [][]byte
		keyz = GetIndexKeysForEvent(ev, ser)
		for _, k := range keyz {
			// log.T.F("index key %x", k)
			if err = txn.Set(k, nil); chk.D(err) {
				return err
			}
		}
		// initialise access counter key
		counterKey := GetCounterKey(&ev.ID, ser)
		// counter value is a 64 bit timestamp and a 32 bit size value access
		//
		// counting means adding current time to this timestamp and dividing by
		// 2 so that frequent accesses have higher, closer to now timestamps for
		// the garbage collection
		//
		// the size value is stored here so that the garbage collector only
		// needs to refer to this type of record to compute a prune operation
		// val := make([]byte, 12)
		// now := uint64(timestamp.Now())
		// size := uint32(len(bin))
		// log.T.Ln("creating access counter record", ev.ID.String(), "timestamp",
		// 	now, "event data size", size)
		// binary.BigEndian.PutUint64(val[:8], now)
		// binary.BigEndian.PutUint32(val[8:], size)
		val := keys.Write(createdat.New(timestamp.Now()), sizer.New(len(bin)))
		log.T.F("counter %x %x", counterKey, val)
		if err = txn.Set(counterKey, val); chk.D(err) {
			return err
		}
		log.T.F("event saved")
		return nil
	})
}
