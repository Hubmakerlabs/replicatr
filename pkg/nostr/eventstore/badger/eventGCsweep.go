package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/del"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	badger2 "github.com/dgraph-io/badger/v4"
	"github.com/minio/sha256-simd"
)

// SweepL1Only implements a simple prune that deletes an event and all related
// indexes.
func (b *Backend) SweepL1Only(serials del.Items) (err error) {
	err = b.DB.Update(func(txn *badger2.Txn) (err error) {
		log.I.Ln("prune with no L2")
		it := txn.NewIterator(badger2.DefaultIteratorOptions)
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
	})
	return
}

// SweepHasL2 implements a prune where only events and access records are
// deleted. This way searches can find them and fetch them from an L2 event
// store.
func (b *Backend) SweepHasL2(serials del.Items) (err error) {
	for i := range serials {
		err = b.DB.Update(func(txn *badger2.Txn) (err error) {
			// An L2 is being used, for this, we replace the encoded raw event record with
			// only the binary version of the eventid.T and zero the size value in the
			// counter key
			prf := index.Event.Key(serial.New(serials[i]))
			it := txn.NewIterator(badger2.IteratorOptions{Prefix: prf})
			defer it.Close()
			it.Seek(prf)
			if it.ValidForPrefix(prf) {
				var v []byte
				v, err = it.Item().ValueCopy(nil)
				if len(v) == sha256.Size {
					log.E.F("event %0x already pruned", v)
					return
				}
				var ev *event.T
				if ev, err = nostrbinary.Unmarshal(v); chk.E(err) {
					return
				}
				if err = txn.Set(prf, ev.ID.Bytes()); chk.E(err) {
					return
				}
			}
			return
		})
		chk.E(err)
	}

	// there is nothing that can be done about database errors at this point anyway.
	err = nil
	return
}

// EventGCSweep implements the EventGCSweep function. If hasL2 is true, a separate prune
// function is called.
func (b *Backend) EventGCSweep(serials del.Items) (err error) {
	if !b.HasL2 {
		err = b.SweepL1Only(serials)
		chk.E(err)
		return
	}
	err = b.SweepHasL2(serials)
	chk.E(err)
	return
}
