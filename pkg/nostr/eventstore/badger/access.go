package badger

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
)

type AccessEvent struct {
	EvID *eventid.T
	Ser  []byte
}

func (a AccessEvent) String() (s string) {
	s = fmt.Sprintf("%s %16x", a.EvID.String(), a.Ser)
	return
}

// IncrementAccesses takes a list of event IDs of events that were accessed in a
// query and updates their access counter records.
func (b *Backend) IncrementAccesses(txMx *sync.Mutex, acc []AccessEvent) (err error) {

out:
	for {
		txMx.Lock()
		err = b.Update(func(txn *badger.Txn) error {
			for i := range acc {
				var item *badger.Item
				key := GetCounterKey(acc[i].EvID, acc[i].Ser)
				v := make([]byte, 12)
				now := timestamp.Now().U64()
				if item, err = txn.Get(key); !chk.E(err) {
					chk.E(item.Value(func(val []byte) error {
						// copy current value to new value
						copy(v, val)
						// update value by averaging current timestamp
						// with stored timestamp so more frequent
						// accesses have a stamp closer to the present
						// only the timestamp is altered, the record
						// size doesn't change
						binary.BigEndian.PutUint64(v[:8], now)
						return nil
					}))
				} else {
					log.I.Ln("creating new access record for", acc[i])
					// timestamp first
					binary.BigEndian.PutUint64(v[:8], now)
					counterKey := index.Counter.Key(serial.New(acc[i].Ser))
					if item, err = txn.Get(counterKey); !chk.E(err) {
						var val []byte
						val, err = item.ValueCopy(nil)
						// then size of value as it wasn't known
						binary.BigEndian.PutUint32(v[8:], uint32(len(val)))
					}
				}
				if err = txn.Set(key, v); chk.E(err) {
					continue
				}
				log.T.Ln("last access for", acc[i], "to", now)
			}
			return nil
		})
		txMx.Unlock()
		// retry if we failed, usually a txn conflict
		if err == nil {
			break out
		}
	}
	return
}
