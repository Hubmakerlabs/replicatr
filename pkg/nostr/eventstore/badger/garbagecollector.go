package badger

import (
	"bytes"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
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
			log.I.Ln("running GC check")
			if err = b.GCRun(); chk.E(err) {

			}
		}
	}
	log.I.Ln("closing badger event store garbage collector")
}

func (b *Backend) GCRun() (err error) {
	log.T.Ln("running garbage collector check")
	var deleteItems DeleteItems
	if deleteItems, err = b.GCCount(); chk.E(err) {
		return
	}
	log.I.Ln(deleteItems)
	if len(deleteItems) < 1 {
		return
	}
	log.I.Ln("deleting:", deleteItems)
	if err = b.Delete(deleteItems); chk.E(err) {
		return
	}
	return
}

// MatchSerial compares a key bytes to a serial, all indexes have the serial at
// the end indicating the event record they refer to, and if they match returns
// true.
func MatchSerial(index, ser []byte) bool {
	l := len(index)
	if l < serial.Len {
		return false
	}
	return bytes.Compare(index[l-serial.Len:], ser) == 0
}

// BadgerDelete implements the Delete function for the case of only using the
// badger.Backend. This removes the event and all indexes.
func (b *Backend) BadgerDelete(serials DeleteItems) (err error) {
	err = b.Update(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for i := range serials {
			if err = txn.Delete(serials[i]); chk.E(err) {
				log.I.Ln("delete failed", serials[i])
				return
			}
		}

		// for it.Rewind(); it.Valid(); it.Next() {
		// 	k := it.Item().Key()
		// 	// check if key matches any of the serials
		// 	for i := range serials {
		// 		if MatchSerial(k, serials[i]) {
		// 			if err = txn.Delete(k); chk.E(err) {
		// 				log.I.Ln(k, serials[i])
		// 				return
		// 			}
		// 		}
		// 	}
		// }
		return
	})
	chk.E(err)
	log.I.Ln("completed prune")
	return
}
