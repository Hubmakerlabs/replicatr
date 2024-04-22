package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/sizer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
	"github.com/minio/sha256-simd"
)

func (b *Backend) SaveEvent(c context.T, ev *event.T) (err error) {
	return b.Update(func(txn *badger.Txn) (err error) {
		// make sure Close waits for this to complete
		b.WG.Add(1)
		defer b.WG.Done()
		// query event by id to ensure we don't save duplicates
		prefix := index.Id.Key(id.New(ev.ID))
		it := txn.NewIterator(badger.IteratorOptions{})
		defer it.Close()
		it.Seek(prefix)
		var foundSerial []byte
		if it.ValidForPrefix(prefix) {
			// event exists - check if it is a stub
			var k []byte
			// get the serial
			if k = it.Item().Key(); chk.E(err) {
				return
			}
			ser := serial.New(nil)
			// copy serial out
			keys.Read(k, id.New(""), ser)
			// save into foundSerial
			foundSerial = ser.Val
			// retrieve the event record
			var item2 *badger.Item
			if item2, err = txn.Get(keys.Write(index.New(index.Event), ser)); !chk.E(err) {
				log.D.Ln("restoring pruned record")
				// we are restoring a pruned event
				var v []byte
				if v, err = item2.ValueCopy(nil); chk.E(err) {
					return
				}
				vLen := len(v)
				if vLen != sha256.Size {
					// not a stub, we already have it
					return eventstore.ErrDupEvent
				}
				// we only need to restore the event binary and bump the access counter key
				// encode to binary
				var bin []byte
				if bin, err = nostrbinary.Marshal(ev); chk.D(err) {
					return
				}
				if err = txn.Set(item2.Key(), bin); chk.D(err) {
					return
				}
				// bump counter key
				counterKey := GetCounterKey(&ev.ID, ser.Val)
				val := keys.Write(createdat.New(timestamp.Now()), sizer.New(vLen))
				log.T.F("counter %x %x", counterKey, val)
				if err = txn.Set(counterKey, val); chk.D(err) {
					return
				}
				// all done, we can return
				return
			}
			// if the ID key was found but not the raw event then we should write it anyway
			log.E.Ln("event ID key but event record missing, writing event and indexes again")
		}
		// encode to binary
		var bin []byte
		if bin, err = nostrbinary.Marshal(ev); chk.D(err) {
			return
		}
		// if the index ID was found but not the raw event, rewrite it and its indexes
		// anyway.
		var idx, ser []byte
		if len(foundSerial) > 0 {
			idx = index.Event.Key(serial.New(foundSerial))
			ser = foundSerial
		} else {
			idx, ser = b.SerialKey()
		}
		// raw event store
		if err = txn.Set(idx, bin); chk.D(err) {
			return
		}
		var keyz [][]byte
		keyz = GetIndexKeysForEvent(ev, ser)
		for _, k := range keyz {
			if err = txn.Set(k, nil); chk.D(err) {
				return
			}
		}
		// initialise access counter key
		counterKey := GetCounterKey(&ev.ID, ser)
		val := keys.Write(createdat.New(timestamp.Now()), sizer.New(len(bin)))
		log.T.F("counter %x %x", counterKey, val)
		if err = txn.Set(counterKey, val); chk.D(err) {
			return
		}
		log.T.F("event saved")
		return
	})
}
