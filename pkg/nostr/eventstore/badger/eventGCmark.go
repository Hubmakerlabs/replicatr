package badger

import (
	"encoding/binary"
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/del"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/count"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/units"
	"github.com/dgraph-io/badger/v4"
	"github.com/minio/sha256-simd"
)

func (b *Backend) EventGCCount() (countItems count.Items, total int, err error) {
	key := make([]byte, index.Len+serial.Len)
	// first find all the non-pruned events.
	if err = b.DB.View(func(txn *badger.Txn) (err error) {
		prf := []byte{byte(index.Event)}
		it := txn.NewIterator(badger.IteratorOptions{
			Prefix: prf,
		})
		for it.Rewind(); it.ValidForPrefix(prf); it.Next() {
			item := it.Item()
			item.KeyCopy(key)
			ser := serial.FromKey(key)
			// skip already pruned items
			size := uint32(item.ValueSize())
			if size == sha256.Size {
				continue
			}
			countItems = append(countItems, &count.Item{Serial: ser.Uint64(), Size: size})
		}
		it.Close()
		return
	}); chk.E(err) {
		// there is nothing that can be done about database errors here so ignore
		err = nil
	}
	v := make([]byte, createdat.Len)
	// second get the datestamps of the items
	if err = b.DB.View(func(txn *badger.Txn) (err error) {
		prf := index.Counter.Key()
		it := txn.NewIterator(badger.IteratorOptions{
			Prefix: prf,
		})
	fresh:
		for it.Rewind(); it.ValidForPrefix(prf); it.Next() {
			item := it.Item()
			item.KeyCopy(key)
			ser := serial.FromKey(key)
			s64 := ser.Uint64()
			for i := range countItems {
				if countItems[i].Serial == s64 {
					// get the value
					if _, err = item.ValueCopy(v); chk.E(err) {
						continue fresh
					}
					countItems[i].Freshness = timestamp.FromUnix(int64(binary.BigEndian.Uint64(v)))
					break
				}
			}
		}
		it.Close()
		return
	}); chk.E(err) {
		// there is nothing that can be done about database errors here so ignore
		err = nil
	}
	total = countItems.Total()
	log.D.F("%d records; total size of data %0.6f MB %0.3f KB high water %0.3f Mb",
		len(countItems),
		float64(total)/units.Mb, float64(total)/units.Kb,
		float64(b.DBHighWater*b.DBSizeLimit/100)/units.Mb)
	return
}

// EventGCMark scans for counter entries and based on GC parameters returns a list
// of the serials of the events that need to be pruned.
func (b *Backend) EventGCMark() (deleteItems del.Items, err error) {
	var countItems count.Items
	var total int
	if countItems, total, err = b.EventGCCount(); chk.E(err) {
		return
	}
	if total < b.DBHighWater*b.DBSizeLimit/100 {
		return
	}
	sort.Sort(countItems)
	pruneOff := total - b.DBLowWater*b.DBSizeLimit/100
	log.T.Ln("will delete nearest to", pruneOff,
		"bytes of events from the event store from the most stale")
	var cumulative, lastIndex int
	for lastIndex = range countItems {
		if cumulative > pruneOff {
			break
		}
		cumulative += int(countItems[lastIndex].Size)
		v := make([]byte, serial.Len)
		binary.BigEndian.PutUint64(v, countItems[lastIndex].Serial)
		deleteItems = append(deleteItems, v)
	}
	sort.Sort(deleteItems)
	log.D.Ln("found", lastIndex,
		"events to prune, which will bring current utilization down to",
		total-cumulative)
	return
}
