package badger

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
)

type AccessEvent struct {
	EvID eventid.T
	Ser  string
}

func MakeAccessEvent(EvID eventid.T, Ser string) (ae *AccessEvent) {
	return &AccessEvent{EvID, Ser}
}

func (a AccessEvent) String() (s string) {
	return fmt.Sprintf("%s %16x", a.EvID.String(), a.Ser)
}

// IncrementAccesses takes a list of event IDs of events that were accessed in a
// query and updates their access counter records.
func (b *Backend) IncrementAccesses(txMx *sync.Mutex, acc []*AccessEvent) (err error) {

out:
	for {
		txMx.Lock()
		err = b.Update(func(txn *badger.Txn) error {
			for i := range acc {
				var item *badger.Item
				key := GetCounterKey(&acc[i].EvID, []byte(acc[i].Ser))
				v := make([]byte, 12)
				now := timestamp.Now().U64()
				if item, err = txn.Get(key); !chk.E(err) {
					if _, err = item.ValueCopy(v); chk.E(err) {
						continue
					}
					// update access record
					binary.BigEndian.PutUint64(v[:8], now)
					if err = txn.Set(key, v); chk.E(err) {
						continue
					}
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
func (b *Backend) AccessLoop(c context.T, txMx *sync.Mutex, accessChan chan *AccessEvent) {
	var accesses []*AccessEvent
	b.WG.Add(1)
	defer b.WG.Done()
	for {
		select {
		case <-c.Done():
			if len(accesses) > 0 {
				log.T.Ln("accesses", accesses)
				chk.E(b.IncrementAccesses(txMx, accesses))
			}
			return
		case acc := <-accessChan:
			log.T.F("adding access to %s %0x", acc.EvID, acc.Ser)
			accesses = append(accesses, &AccessEvent{acc.EvID, acc.Ser})
		}
	}
}
