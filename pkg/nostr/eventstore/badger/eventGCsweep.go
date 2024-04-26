package badger

import (
	"encoding/binary"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/del"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	badger2 "github.com/dgraph-io/badger/v4"
)

// SweepL1Only implements a simple prune that deletes an event and all related
// indexes.
func (b *Backend) SweepL1Only(serials del.Items) (err error) {
	err = b.DB.Update(func(txn *badger2.Txn) (err error) {
		log.I.Ln("prune with no L2")
		it := txn.NewIterator(badger2.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().Key()
			// check if key matches any of the serials
			for i := range serials {
				if serial.Match(k, serials[i]) {
					if err = txn.Delete(k); chk.E(err) {
						log.I.Ln(k, serials[i])
						return
					}
					break
				}
			}
		}
		return
	})
	return
}

// SweepHasL2 implements a prune where only events and access records are
// deleted. This way searches can find them and fetch them from an L2 event
// store.
func (b *Backend) SweepHasL2(serials del.Items) (err error) {
	err = b.DB.Update(func(txn *badger2.Txn) (err error) {
		// An L2 is being used, for this, we replace the encoded raw event record with
		// only the binary version of the eventid.T and zero the size value in the
		// counter key
		it := txn.NewIterator(badger2.DefaultIteratorOptions)
		for {
			for it.Rewind(); it.Valid(); it.Next() {
				k := it.Item().Key()
				if k[0] != index.Event.Byte() && k[0] != index.Counter.Byte() {
					continue
				}
				// check if key matches any of the serials
				for i := range serials {
					if !serial.Match(k, serials[i]) {
						continue
					}
					switch k[0] {
					case index.Event.Byte():
						log.T.Ln("deleting event serial", binary.BigEndian.Uint64(k[1:]))
						// replace the value with the event ID as binary.
						var v []byte
						if v, err = it.Item().ValueCopy(nil); chk.E(err) {
							continue
						}
						if len(v) == 32 {
							log.W.F("pruning item that is already pruned %d",
								binary.BigEndian.Uint64(serials[i]))
							continue
						}
						var evt *event.T
						if evt, err = nostrbinary.Unmarshal(v); chk.E(err) {
							continue
						}
						log.T.F("replacing event with the event ID %0x", evt.ID.Bytes())
						// set the value of the key to the event id hash as binary
						if err = txn.Set(k, evt.ID.Bytes()); chk.E(err) {
							continue
						}
						if err != nil {
							log.E.F("error replacing event with the event ID %0x, %s",
								evt.ID.Bytes(), err)
							continue
						}
					case index.Counter.Byte():
						log.T.Ln("zeroing counter serial",
							binary.BigEndian.Uint64(serials[i]))
						var v []byte
						if v, err = it.Item().ValueCopy(nil); chk.E(err) {
							continue
						}
						// zero out the size.
						copy(v[8:12], make([]byte, 4))
						if err = txn.Set(k, v); chk.E(err) {
							continue
						}
					}
				}
			}
			if err == nil {
				break
			}
		}
		it.Close()
		return
	})
	chk.E(err)
	// there is nothing that can be done about database errors at this point anyway.
	err = nil
	return
}

// EventGCSweep implements the EventGCSweep function. If hasL2 is true, a separate prune
// function is called.
func (b *Backend) EventGCSweep(serials del.Items) (err error) {
	if !b.HasL2 {
		err = b.SweepL1Only(serials)
		chk.E(err)
		return
	}
	err = b.SweepHasL2(serials)
	chk.E(err)
	return
}
