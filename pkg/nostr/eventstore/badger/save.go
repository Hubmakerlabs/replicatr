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
		prf := index.Id.Key(id.New(ev.ID))
		it := txn.NewIterator(badger.IteratorOptions{})
		defer it.Close()
		it.Seek(prf)
		var foundSerial []byte
		seri := serial.New(nil)
		if it.ValidForPrefix(prf) {
			// event exists - check if it is a stub
			var k []byte
			// get the serial
			k = it.Item().Key()
			// log.I.F("id key found %0x", k)
			// copy serial out
			keys.Read(k, index.Empty(), id.New(""), seri)
			// save into foundSerial
			foundSerial = seri.Val
		}
		if foundSerial != nil && len(foundSerial) > 0 {
			// log.I.F("found serial %d", binary.BigEndian.Uint64(foundSerial))
			// retrieve the event record
			evKey := keys.Write(index.New(index.Event), seri)
			it2 := txn.NewIterator(badger.IteratorOptions{})
			defer it2.Close()
			it2.Seek(evKey)
			if it2.ValidForPrefix(evKey) {
				// log.I.F("found event key %0x", evKey)
				// we are restoring a pruned event
				var v []byte
				if v, err = it2.Item().ValueCopy(nil); chk.E(err) {
					return
				}
				vLen := len(v)
				if vLen != sha256.Size {
					// log.D.F("not pruned record %0x length %d", evKey, vLen)
					// not a stub, we already have it
					return eventstore.ErrDupEvent
				}
				// we only need to restore the event binary and write the access counter key
				// encode to binary
				var bin []byte
				if bin, err = nostrbinary.Marshal(ev); chk.D(err) {
					return
				}
				if err = txn.Set(it2.Item().Key(), bin); chk.E(err) {
					return
				}
				// bump counter key
				counterKey := GetCounterKey(&ev.ID, seri.Val)
				val := keys.Write(createdat.New(timestamp.Now()), sizer.New(len(bin)))
				// log.D.F("counter %x %x", counterKey, val)
				if err = txn.Set(counterKey, val); chk.E(err) {
					return
				}
				// log.D.F("restored pruned record %s", ev.ID)
				// all done, we can return
				return
			} else {
				return eventstore.ErrDupEvent
			}
		}
		var idx, ser []byte
		if len(foundSerial) > 0 {
			// if the ID key was found but not the raw event then we should write it anyway
			log.E.F("event ID %s found key but event record missing, rewriting event and indexes",
				ev.ID)
			idx = index.Event.Key(serial.New(foundSerial))
			ser = foundSerial
		} else {
			idx, ser = b.SerialKey()
		}
		// encode to binary
		var bin []byte
		if bin, err = nostrbinary.Marshal(ev); chk.E(err) {
			return
		}
		// raw event store
		if err = txn.Set(idx, bin); chk.E(err) {
			return
		}
		var keyz [][]byte
		keyz = GetIndexKeysForEvent(ev, ser)
		for _, k := range keyz {
			if err = txn.Set(k, nil); chk.E(err) {
				return
			}
		}
		// initialise access counter key
		counterKey := GetCounterKey(&ev.ID, ser)
		val := keys.Write(createdat.New(timestamp.Now()), sizer.New(len(bin)))
		log.T.F("counter %0x %0x", counterKey, val)
		if err = txn.Set(counterKey, val); chk.E(err) {
			return
		}
		log.T.F("event saved")
		return
	})
}
