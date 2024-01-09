package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"

	nostr_binary "github.com/Hubmakerlabs/replicatr/pkg/nostr/binary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/relay/eventstore"
	"github.com/dgraph-io/badger/v4"
)

func (b *Backend) SaveEvent(c context.T, evt *event.T) (e error) {
	return b.Update(func(txn *badger.Txn) (e error) {
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
		var bin []byte
		if bin, e = nostr_binary.Marshal(evt); log.E.Chk(e) {
			return
		}
		idx := b.Serial()
		// raw event store
		if e = txn.Set(idx, bin); log.E.Chk(e) {
			return
		}
		for _, k := range getIndexKeysForEvent(evt, idx[1:]) {
			if e = txn.Set(k, nil); log.E.Chk(e) {
				return
			}
		}

		return nil
	})
}
