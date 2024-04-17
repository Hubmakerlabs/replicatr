package badger

import (
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/sizer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/del"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/keys/count"
	"github.com/dgraph-io/badger/v4"
)

// BadgerGCCount scans for counter entries and based on GC parameters returns a list
// of the serials of the events that need to be pruned.
func (b *Backend) BadgerGCCount() (deleteItems del.Items, err error) {
	var countItems count.Items
	v := make([]byte, createdat.Len+sizer.Len)
	key := make([]byte, index.Len+id.Len+serial.Len)
	err = b.View(func(txn *badger.Txn) (err error) {
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
			countItems = append(countItems, count.MakeItem(ser, ts, size))
		}
		it.Close()
		return
	})
	if chk.E(err) {
		return
	}
	total := countItems.Total()
	log.I.F("total size of data %0.6f MB %0.3f KB high water %0.3f Mb",
		float64(total)/Megabyte, float64(total)/1000,
		float64(b.DBHighWater*b.DBSizeLimit/100)/Megabyte)
	if total < b.DBHighWater*b.DBSizeLimit/100 {
		return
	}
	// log.W.Ln("GC needs to run")
	sort.Sort(countItems)
	pruneOff := total - b.DBLowWater*b.DBSizeLimit/100
	log.I.Ln("will delete nearest to", pruneOff,
		"bytes of events from the event store from the most stale")
	var cumulative, lastIndex int
	for lastIndex = range countItems {
		if cumulative > pruneOff {
			break
		}
		cumulative += int(countItems[lastIndex].Size)
		deleteItems = append(deleteItems, countItems[lastIndex].Serial)
	}
	sort.Sort(deleteItems)
	log.I.Ln("found", lastIndex,
		"events to prune, which will bring current utilization down to",
		total-cumulative)
	return
}
