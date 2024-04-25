package badger

import (
	"encoding/binary"
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/del"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/sizer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
	"github.com/minio/sha256-simd"
)

type IndexCount struct {
	keySize  uint16 // keys are pretty small
	valSize  uint32 // values can be up to half a megabyte default limit
	accessed timestamp.T
}

type prunedItem struct {
	ser      uint64
	accessed timestamp.T
	size     uint32
}

type SortPruned []prunedItem

func (s SortPruned) Len() int           { return len(s) }
func (s SortPruned) Less(i, j int) bool { return s[i].accessed < s[j].accessed }
func (s SortPruned) Swap(i, j int)      {}

const prunedSize = sha256.Size + createdat.Len + sizer.Len

func IndexGCCount() func(bi any) (deleteItems del.Items, err error) {
	return func(bi any) (deleteItems del.Items, err error) {
		b, ok := bi.(*Backend)
		if !ok {
			err = log.E.Err("backend type does not match badger eventstore")
			return
		}
		serials := make(map[uint64]IndexCount)
		err = b.DB.View(func(txn *badger.Txn) (err error) {
			// first build the map of event record serials and whether they have been
			// pruned.
			it := txn.NewIterator(badger.IteratorOptions{})
			for it.Rewind(); it.Valid(); it.Next() {
				keySize := it.Item().KeySize()
				valSize := it.Item().ValueSize()
				k := it.Item().Key()
				ser := binary.BigEndian.Uint64(serial.FromKey(k).Val)
				s, exists := serials[ser]
				var access timestamp.T
				// if it is a counter, get the access time to add to the record
				if k[0] == index.Counter.Byte() {
					// get the access time
					var v []byte
					if v, err = it.Item().ValueCopy(nil); chk.E(err) {
						// what to do? this shouldn't happen, perhaps later build a "repair database"
						// function that regenerates everything from events and wipes out all pruned
						// indexes.
					}
					accessed := createdat.New(0)
					keys.Read(v, accessed, sizer.New(0))
					access = accessed.Val
				}
				if exists {
					s.valSize += uint32(valSize)
					s.keySize += uint16(keySize)
					if access > 0 {
						s.accessed = access
					}
				} else {
					ic := IndexCount{
						keySize: uint16(keySize),
						valSize: uint32(valSize),
					}
					if access > 0 {
						ic.accessed = access
					}
					serials[ser] = ic
				}
			}
			// we can close this iterator because we are done with it.
			it.Close()

			return
		})
		chk.E(err)
		// there is currently nothing that can be done about database errors anyway
		err = nil
		// we can know whether a record is pruned or not by the cumulative size of its
		// valSize... if it is an event ID hash + timestamp + size (44), that means it's
		// pruned.
		var deletes []uint64
		var totalSize uint64
		for i := range serials {
			if serials[i].valSize == prunedSize {
				// count the total as we go
				totalSize += uint64(serials[i].valSize) + uint64(serials[i].keySize)
			} else {
				deletes = append(deletes, i)
			}
		}
		// we have all the live records, so we can delete them now, and after this
		// function they should be garbage collected.
		for i := range deletes {
			delete(serials, deletes[i])
		}
		// next, assemble the pruned records into a slice so they can be sorted by
		// timestamp
		var sorted SortPruned
		for i := range serials {
			sorted = append(sorted, prunedItem{
				ser:      i,
				size:     serials[i].valSize + uint32(serials[i].keySize),
				accessed: serials[i].accessed,
			})
		}
		sort.Sort(sorted)
		// now we have totalSize and the pruned events in order, so we can now calculate
		// a prune based on the GC max size and high water mark, which we will say is
		// the headroom that indexes can occupy.
		//
		// the amount of storage between the high water and size limit can be filled
		// with indexes, anything that pops over top of that needs to be removed, down
		// to the proportion of limit/high water. it grows slower but this simplifies
		// how to specify limits.
		headroom := b.DBSizeLimit * b.DBHighWater / 100 * b.DBHighWater / 100
		deletable := int(totalSize) - headroom
		if deletable < 0 {
			// nothing to do
			log.I.Ln("indexes are not exceeding headroom")
			return
		}
		log.I.F("need to delete at least %d bytes of indexes", deletable)
		var lastIndex, totalMarked int
		for lastIndex = range sorted {
			totalMarked += int(sorted[lastIndex].size)
			if totalMarked > deletable {
				break
			}
		}
		// now we have the oldest indexes that compose the nearest to the amount that
		// needs to be pruned.
		log.I.F("found %d indexes to prune which will reduce index space usage by %d",
			lastIndex, totalMarked)
		deleteItems = make([][]byte, 0, lastIndex+1)
		for i := 0; i <= lastIndex; i++ {
			item := make([]byte, serial.Len)
			binary.BigEndian.PutUint64(item, sorted[i].ser)
			deleteItems = append(deleteItems, item)
		}
		return
	}
}
