package badger

import (
	"time"

	serial2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/del"
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
	if deleteItems, err = GcCount(b); chk.E(err) {
		return
	}
	if len(deleteItems) < 1 {
		return
	}
	log.I.Ln("deleting:", deleteItems)
	if err = b.Prune(b, deleteItems); chk.E(err) {
		return
	}
	return
}

// Prune implements the Prune function for the case of only using the
// badger.Backend. This removes the event and all indexes.
func Prune(bi any, serials del.Items) (err error) {
	b, ok := bi.(*Backend)
	if !ok {
		err = log.E.Err("backend type does not match badger eventstore")
		return
	}
	err = b.Update(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().Key()
			// check if key matches any of the serials
			for i := range serials {
				if serial2.Match(k, serials[i]) {
					if err = txn.Delete(k); chk.E(err) {
						log.I.Ln(k, serials[i])
						return
					}
					break
				}
			}
		}
		return
	})
	chk.E(err)
	log.T.Ln("completed prune")
	chk.E(b.DB.Sync())
	return
}
