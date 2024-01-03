package badger

import (
	"context"
	"encoding/hex"

	"github.com/Hubmakerlabs/replicatr/pkg/eventstore"
	"github.com/dgraph-io/badger/v4"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr"
	nostr_binary "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/binary"
)

func (b *BadgerBackend) SaveEvent(ctx context.Context, evt *nostr.Event) (e error) {
	return b.Update(func(txn *badger.Txn) (e error) {
		// query event by id to ensure we don't save duplicates
		id, _ := hex.DecodeString(evt.ID)
		prefix := make([]byte, 1+8)
		prefix[0] = indexIdPrefix
		copy(prefix[1:], id)
		it := txn.NewIterator(badger.IteratorOptions{})
		defer it.Close()
		it.Seek(prefix)
		if it.ValidForPrefix(prefix) {
			// event exists
			return eventstore.ErrDupEvent
		}

		// encode to binary
		bin, err := nostr_binary.Marshal(evt)
		if err != nil {
			return err
		}

		idx := b.Serial()
		// raw event store
		if err := txn.Set(idx, bin); err != nil {
			return err
		}

		for _, k := range getIndexKeysForEvent(evt, idx[1:]) {
			if err := txn.Set(k, nil); err != nil {
				return err
			}
		}

		return nil
	})
}
