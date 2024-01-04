package badger

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

func (b *BadgerBackend) runMigrations() (e error) {
	return b.Update(func(txn *badger.Txn) (e error) {
		var version uint16

		var item *badger.Item
		item, e = txn.Get([]byte{dbVersionKey})
		if errors.Is(e, badger.ErrKeyNotFound) {
			version = 0
		} else if e != nil {
			return
		} else {
			log.E.Chk(item.Value(func(val []byte) (e error) {
				version = binary.BigEndian.Uint16(val)
				return nil
			}))
		}

		// do the migrations in increasing steps (there is no rollback)
		//

		// the 3 first migrations go to trash because on version 3 we need to export and import all the data anyway
		if version < 3 {
			// if there is any data in the relay we will stop and notify the user,
			// otherwise we just set version to 3 and proceed
			prefix := []byte{indexIdPrefix}
			it := txn.NewIterator(badger.IteratorOptions{
				PrefetchValues: true,
				PrefetchSize:   100,
				Prefix:         prefix,
			})
			defer it.Close()

			hasAnyEntries := false
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				hasAnyEntries = true
				break
			}

			if hasAnyEntries {
				return fmt.Errorf("your database is at version %d, but " +
					"in order to migrate up to version 3 you must " +
					"manually export all the events and then import " +
					"again: run an old version of this software, export " +
					"the data, then delete the database files, run the " +
					"new version, import the data back in", version)
			}

			log.E.Chk(b.bumpVersion(txn, 3))
		}

		if version < 4 {
			// ...
		}

		return nil
	})
}

func (b *BadgerBackend) bumpVersion(txn *badger.Txn, version uint16) (e error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, version)
	return txn.Set([]byte{dbVersionKey}, buf)
}
