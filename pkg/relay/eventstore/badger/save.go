package badger

import (
	"context"
	nostr_binary "github.com/Hubmakerlabs/replicatr/pkg/nostr/binary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/dgraph-io/badger/v4"
	"github.com/fiatjaf/eventstore"
)

func (b *Backend) SaveEvent(ctx context.Context, evt *nip1.Event) error {
	return b.Update(func(txn *badger.Txn) error {
		// query event by id to ensure we don't save duplicates
		id := evt.ID.Bytes()
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
