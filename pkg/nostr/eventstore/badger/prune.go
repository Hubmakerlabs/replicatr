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

// Prune implements the Prune function. If hasL2 is true, the function
func Prune(hasL2 bool) func(bi any, serials del.Items) (err error) {
	return func(bi any, serials del.Items) (err error) {
		b, ok := bi.(*Backend)
		if !ok {
			err = log.E.Err("backend type does not match badger eventstore")
			return
		}
		err = b.Update(func(txn *badger2.Txn) (err error) {
			if !hasL2 {
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
			}
			// log.I.Ln("prune with L2")
			// An L2 is being used, for this, we replace the encoded raw event record with
			// only the binary version of the eventid.T and zero the size value in the
			// counter key
			it := txn.NewIterator(badger2.DefaultIteratorOptions)
			for {
				for it.Rewind(); it.Valid(); it.Next() {
					k := it.Item().Key()
					// todo: wth do we do if the txn.Set functions fail??? they can't be retried? they shouldn't fail?
					switch k[0] {
					case index.Event.Byte(), index.Counter.Byte():
						// check if key matches any of the serials
						for i := range serials {
							if serial.Match(k, serials[i]) {
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
									log.T.Ln("deleting counter serial", binary.BigEndian.Uint64(k[9:]))
									// delete access record so it isn't counted.
									// log.T.F("deleting counter record counter %0x", k)
									if err = txn.Delete(k); chk.E(err) {
										continue
									}
								}
							}
						}
					default:
					}
				}
				if err == nil {
					break
				}
			}
			it.Close()
			return
		})
		// chk.E(b.DB.Sync())
		chk.E(err)
		// log.D.Ln("completed prune")
		// b.GCCount()
		return
	}
}
