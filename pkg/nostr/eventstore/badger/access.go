package badger

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
)

type AccessEvent struct {
	EvID eventid.T
	Ts   timestamp.T
	Ser  *serial.T
}

// MakeAccessEvent generates an *AccessEvent from an event ID and serial.
func MakeAccessEvent(EvID eventid.T, Ser *serial.T) (ae *AccessEvent) {
	return &AccessEvent{EvID, timestamp.Now(), Ser}
}

func (a AccessEvent) String() (s string) {
	return fmt.Sprintf("[%s, %v, %d]", a.EvID.String(), a.Ts.Time(), a.Ser.Uint64())
}

// IncrementAccesses takes a list of event IDs of events that were accessed in a
// query and updates their access counter records.
func (b *Backend) IncrementAccesses(acc *AccessEvent) (err error) {
out:
	for {
		err = b.Update(func(txn *badger.Txn) (err error) {
			key := GetCounterKey(acc.Ser)
			it := txn.NewIterator(badger.IteratorOptions{})
			defer it.Close()
			if it.Seek(key); it.ValidForPrefix(key) {
				// update access record
				if err = txn.Set(key, acc.Ts.Bytes()); chk.E(err) {
					return
				}
			}
			log.T.Ln("last access for", acc.Ser.Uint64(), acc.Ts.U64())
			return nil
		})
		// retry if we failed, usually a txn conflict
		if err == nil {
			break out
		}
	}
	return
}

// AccessLoop is meant to be run as a goroutine to gather access events in a
// query and when it finishes, bump all the access records
func (b *Backend) AccessLoop(c context.T, accCh chan *AccessEvent) {
	b.WG.Add(1)
	defer b.WG.Done()
	for {
		select {
		case <-b.Ctx.Done():
			return
		case <-c.Done():
			return
		case acc := <-accCh:
			if acc == nil {
				// channel has been closed
				return
			}
			chk.E(b.IncrementAccesses(acc))
		}
	}
}
