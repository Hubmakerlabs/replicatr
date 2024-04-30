package badger

import (
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
	"github.com/minio/sha256-simd"
)

type IndexCount struct {
	keySize  uint16 // keys are pretty small
	valSize  uint32 // values can be up to half a megabyte default limit
	accessed timestamp.T
}

type IndexMap map[uint64]*IndexCount

type prunedItem struct {
	ser      uint64
	accessed timestamp.T
	size     uint32
}

type SortPruned []prunedItem

func (s SortPruned) Total() (total int) {
	for i := range s {
		total += int(s[i].size)
	}
	return
}
func (s SortPruned) Len() int           { return len(s) }
func (s SortPruned) Less(i, j int) bool { return s[i].accessed < s[j].accessed }
func (s SortPruned) Swap(i, j int)      {}

func (b *Backend) IndexGCCount() (serials IndexMap, err error) {
	serials = make(IndexMap)
	if err = b.DB.View(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.IteratorOptions{})
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			if len(k) < serial.Len {
				// most likely db ID key or version key
				continue
			}
			ser := serial.FromKey(k)
			uSer := ser.Uint64()
			if _, ok := serials[uSer]; !ok {
				// allocate the entry
				serials[uSer] = &IndexCount{}
			}
			serials[uSer].keySize += uint16(len(k))
			serials[uSer].valSize += uint32(item.ValueSize())
			if k[0] == index.Counter.Byte() {
				var v []byte
				if v, err = item.ValueCopy(nil); chk.E(err) {
					continue
				}
				serials[uSer].accessed = timestamp.FromBytes(v)
			}
		}
		return
	}); chk.E(err) {
		err = nil
	}
	return
}

// IndexGCMark scans all indexes, collects the data related to the set of pruned
// indexes, sorts them by access time and selects the set of index serials to remove
func (b *Backend) IndexGCMark() (toDelete []uint64, err error) {
	var serials IndexMap
	if serials, err = b.IndexGCCount(); chk.E(err) {
		return
	}
	var pruned SortPruned
	for i := range serials {
		// copy record to pruned list
		if serials[i].valSize == sha256.Size+createdat.Len {
			pruned = append(pruned, prunedItem{
				ser:      i,
				accessed: serials[i].accessed,
				size:     serials[i].valSize + uint32(serials[i].keySize),
			})
		}
	}
	headroom := b.DBSizeLimit * (100 - b.DBHighWater) / 100
	lw, hw := headroom*b.DBLowWater/100, headroom*b.DBHighWater/100
	total := pruned.Total()
	log.D.F("total size of pruned indexes %d LW %d HW %d", total, lw, hw)
	if total < hw {
		return
	}
	sort.Sort(pruned)
	var lastIndex int
	space := headroom
	for lastIndex = range pruned {
		if space < lw {
			break
		}
		space -= int(pruned[lastIndex].size)
	}
	log.D.F("deleting %d indexes using %d bytes to bring pruned index size to %d",
		lastIndex+1, headroom-space, space)
	for i := range pruned {
		if i > lastIndex {
			break
		}
		toDelete = append(toDelete, pruned[i].ser)
	}
	return
}
