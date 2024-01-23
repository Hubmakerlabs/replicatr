package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"

	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	nostr_binary "github.com/Hubmakerlabs/replicatr/pkg/nostr/binary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/dgraph-io/badger/v4"
)

func (b *BadgerBackend) SaveEvent(c context.T, evt *event.T) (e error) {
	return b.Update(func(txn *badger.Txn) (e error) {
		b.D.Ln("saving event")
		// query event by id to ensure we don't save duplicates
		id, _ := hex.Dec(evt.ID.String())
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
		b.D.Ln("encoding to binary")
		// encode to binary
		var bin []byte
		if bin, e = nostr_binary.Marshal(evt); b.Fail(e) {
			return e
		}
		b.D.F("binary encoded %x", bin)
		idx := b.Serial()
		// raw event store
		b.D.F("setting event")
		if e = txn.Set(idx, bin); b.Fail(e) {
			return e
		}
		b.D.F("get index keys for event")
		for _, k := range getIndexKeysForEvent(evt, idx[1:]) {
			b.D.F("index key %x", k)
			if e = txn.Set(k, nil); b.Fail(e) {
				return e
			}
		}
		b.D.F("event saved")
		return nil
	})
}
