package app

import (
	"github.com/dgraph-io/badger/v4"
	bdb "mleku.dev/git/nostr/eventstore/badger"
)

// Wipe clears the badgerDB local event store/cache.
func (rl *Relay) Wipe(store *bdb.Backend) (err error) {
	return store.DB.Update(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		var count int
		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().Key()
			if err = txn.Delete(k); !chk.E(err) {
				count++
			}
		}
		it.Close()
		log.I.Ln("deleted", count, "records")
		return
	})
}
