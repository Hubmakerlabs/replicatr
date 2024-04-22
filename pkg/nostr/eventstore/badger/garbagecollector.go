package badger

import (
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/del"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/sizer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
)

// GarbageCollector starts up a ticker that runs a check on space utilisation
// and when it exceeds the high-water mark, prunes back to the low-water mark.
//
// This function should be invoked as a goroutine, and will terminate when the
// backend context is canceled.
func (b *Backend) GarbageCollector() {
	log.W.Ln("starting badger back-end garbage collector:")
	log.I.F("max size %0.3f MB; "+
		"high water %0.3f MB; "+
		"low water %0.3f MB "+
		"(MB = %d bytes) "+
		"GC check frequency %v",
		float32(b.DBSizeLimit)/Megabyte,
		float32(b.DBHighWater*b.DBSizeLimit/100)/Megabyte,
		float32(b.DBLowWater*b.DBSizeLimit/100)/Megabyte,
		Megabyte,
		b.GCFrequency,
	)
	var err error
	if err = b.GCRun(); chk.E(err) {
	}
	gcTicker := time.NewTicker(b.GCFrequency)
out:
	for {
		select {
		case <-b.Ctx.Done():
			log.W.Ln("backend context done")
			break out
		case <-gcTicker.C:
			log.T.Ln("running GC check")
			if err = b.GCRun(); chk.E(err) {

			}
		}
	}
	log.I.Ln("closing badger event store garbage collector")
}

func (b *Backend) GCRun() (err error) {
	log.T.Ln("running garbage collector check")
	var deleteItems del.Items
	if deleteItems, err = b.GCCount(); chk.E(err) {
		return
	}
	if len(deleteItems) < 1 {
		return
	}
	log.I.Ln("deleting:", deleteItems)
	if err = b.Prune(deleteItems); chk.E(err) {
		return
	}
	return
}

// Prune implements the Prune function for the case of only using the
// badger.Backend. This removes the event and all indexes.
func Prune(hasL2 bool) func(bi any, serials del.Items) (err error) {
	return func(bi any, serials del.Items) (err error) {
		b, ok := bi.(*Backend)
		if !ok {
			err = log.E.Err("backend type does not match badger eventstore")
			return
		}
		err = b.Update(func(txn *badger.Txn) (err error) {
			if !hasL2 {
				it := txn.NewIterator(badger.DefaultIteratorOptions)
				defer it.Close()
				for it.Rewind(); it.Valid(); it.Next() {
					k := it.Item().Key()
					// check if key matches any of the serials
					for i := range serials {
						if serial.Match(k, serials[i]) {
							if err = txn.Delete(k); chk.E(err) {
								log.I.Ln(k, serials[i])
								return
							}
							break
						}
					}
				}
				return
			}
			// An L2 is being used, for this, we replace the encoded raw event record with
			// only the binary version of the eventid.T and zero the size value in the
			// counter key
			it := txn.NewIterator(badger.DefaultIteratorOptions)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				k := it.Item().Key()
				// todo: wth do we do if the txn.Set functions fail??? they can't be retried? they shouldn't fail?
				switch k[0] {
				case index.Event.Byte(), index.Counter.Byte():
					// check if key matches any of the serials
					for i := range serials {
						if serial.Match(k, serials[i]) {
							switch k[0] {
							case index.Event.Byte():
								// replace the value with the event ID as binary.
								var v []byte
								if v, err = it.Item().ValueCopy(nil); chk.E(err) {
									continue
								}
								var evt *event.T
								if evt, err = nostrbinary.Unmarshal(v); chk.E(err) {
									// todo: maybe this should mean we delete the record?
									continue
								}
								// set the value of the key to the event id hash as binary
								if err = txn.Set(k, evt.ID.Bytes()); chk.E(err) {
									continue
								}
							case index.Counter.Byte():
								// update access timestamp and set size to zero.
								v := keys.Write(createdat.New(timestamp.Now()), sizer.New(0))
								if err = txn.Set(k, v); chk.E(err) {
									continue
								}
							}
							break
						}
					}
				default:
				}
			}
			return
		})
		chk.E(err)
		log.T.Ln("completed prune")
		chk.E(b.DB.Sync())
		return
	}
}
