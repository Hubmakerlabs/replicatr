package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/dgraph-io/badger/v4"
)

func (b *Backend) IndexGCSweep(toDelete []uint64) (err error) {
	if err = b.DB.Update(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().KeyCopy(nil)
			for i := range toDelete {
				ser := serial.Make(toDelete[i])
				if serial.Match(k, ser) {
					// log.I.Ln("deleting index", toDelete[i], k)
					if err = txn.Delete(k); chk.E(err) {
					}
				}
			}
		}
		return
	}); chk.E(err) {
		err = nil
	}
	return
}
