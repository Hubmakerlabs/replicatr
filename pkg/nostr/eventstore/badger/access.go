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
			// update access record
			if err = txn.Set(key, acc.Ts.Bytes()); chk.E(err) {
				return
			}
			return nil
		})
		// retry if we failed, usually a txn conflict
		if err == nil {
			break out
		}
		log.I.Ln("retrying access bump")
	}
	log.I.Ln("access added to", acc.EvID, b.Path)
	return
}

// AccessLoop is meant to be run as a goroutine to gather access events in a
// query and when it finishes, bump all the access records
func (b *Backend) AccessLoop(c context.T, accCh chan *AccessEvent, f string) {
	log.T.Ln("access loop started", f, b.Path)
	defer log.I.Ln("access loop terminated", f, b.Path)
	// b.WG.Add(1)
	// defer b.WG.Done()
	for {
		log.I.Ln("access loop", f, b.Path)
		select {
		case <-c.Done():
			return
		case <-b.Ctx.Done():
			return
		case acc := <-accCh:
			if acc != nil {
				go chk.E(b.IncrementAccesses(acc))
			} else {
				return
			}
		}
	}
}
