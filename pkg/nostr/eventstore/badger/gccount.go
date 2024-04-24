package badger

import (
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/del"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/count"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/sizer"
	"github.com/Hubmakerlabs/replicatr/pkg/units"
	"github.com/dgraph-io/badger/v4"
)

// GCCount scans for counter entries and based on GC parameters returns a list
// of the serials of the events that need to be pruned.
func GCCount() func(bi any) (deleteItems del.Items, err error) {
	return func(bi any) (deleteItems del.Items, err error) {
		b, ok := bi.(*Backend)
		if !ok {
			err = log.E.Err("backend type does not match badger eventstore")
			return
		}
		var countItems count.Items
		v := make([]byte, createdat.Len+sizer.Len)
		key := make([]byte, index.Len+id.Len+serial.Len)
		// var restSer []uint64
		if err = b.DB.View(func(txn *badger.Txn) (err error) {
			it := txn.NewIterator(badger.IteratorOptions{
				Prefix: count.Prefix,
			})
			for it.Rewind(); it.ValidForPrefix(count.Prefix); it.Next() {
				item := it.Item()
				item.KeyCopy(key)
				ser := serial.FromKey(key)
				if _, err = item.ValueCopy(v); chk.E(err) || len(v) == 0 {
					continue
				}
				ts, size := createdat.New(0), sizer.New(0)
				keys.Read(v, ts, size)
				// skip already pruned items
				if size.Val == 0 {
					log.I.F("found count with zero len %0x", key)
					continue
				}
				countItems = append(countItems, count.MakeItem(ser, ts, size))
			}
			it.Close()
			return
		}); chk.E(err) {
			return
		}
		total := countItems.Total()
		log.I.F("%d records; total size of data %0.6f MB %0.3f KB high water %0.3f Mb",
			len(countItems),
			float64(total)/units.Mb, float64(total)/units.Kb,
			float64(b.DBHighWater*b.DBSizeLimit/100)/units.Mb)
		if total < b.DBHighWater*b.DBSizeLimit/100 {
			return
		}
		// log.W.Ln("GC needs to run")
		sort.Sort(countItems)
		pruneOff := total - b.DBLowWater*b.DBSizeLimit/100
		log.T.Ln("will delete nearest to", pruneOff,
			"bytes of events from the event store from the most stale")
		var cumulative, lastIndex int
		// var serList []uint64
		for lastIndex = range countItems {
			if cumulative > pruneOff {
				// restSer = append(restSer, binary.BigEndian.Uint64(countItems[lastIndex].Serial))
				// continue
				break
			}
			cumulative += int(countItems[lastIndex].Size)
			deleteItems = append(deleteItems, countItems[lastIndex].Serial)
			// serList = append(serList, binary.BigEndian.Uint64(countItems[lastIndex].Serial))
		}
		// log.I.Ln("serList", serList, "restSer", restSer)
		sort.Sort(deleteItems)
		log.I.Ln("found", lastIndex,
			"events to prune, which will bring current utilization down to",
			total-cumulative)
		return
	}
}
