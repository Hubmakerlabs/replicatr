package badger

import (
	"bytes"
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/sizer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/dgraph-io/badger/v4"
)

var counterPrefix = []byte{byte(index.Counter)}

type CountItem struct {
	Serial    []byte
	Size      uint32
	Freshness timestamp.T
}

type CountItems []*CountItem

func MakeCountItem(ser *serial.T, ts *createdat.T,
	size *sizer.T) *CountItem {

	return &CountItem{
		Serial:    ser.Val,
		Freshness: ts.Val,
		Size:      size.Val,
	}
}

func (c CountItems) Len() int           { return len(c) }
func (c CountItems) Less(i, j int) bool { return c[i].Freshness < c[j].Freshness }
func (c CountItems) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c CountItems) Total() (total int) {
	for i := range c {
		total += int(c[i].Size)
	}
	return
}

type DeleteItems [][]byte

func (c DeleteItems) Len() int           { return len(c) }
func (c DeleteItems) Less(i, j int) bool { return bytes.Compare(c[i], c[j]) < 0 }
func (c DeleteItems) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

// GCCount scans for counter entries and based on GC parameters returns a list
// of the serials of the events that need to be pruned.
func (b *Backend) GCCount() (deleteItems DeleteItems, err error) {
	var countItems CountItems
	v := make([]byte, createdat.Len+sizer.Len)
	key := make([]byte, index.Len+id.Len+serial.Len)
	err = b.View(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.IteratorOptions{
			Prefix: counterPrefix,
		})
		for it.Rewind(); it.ValidForPrefix(counterPrefix); it.Next() {
			item := it.Item()
			item.KeyCopy(key)
			ser := serial.FromKey(key)
			if _, err = item.ValueCopy(v); chk.E(err) || len(v) == 0 {
				continue
			}
			ts, size := createdat.New(0), sizer.New(0)
			keys.Read(v, ts, size)
			countItems = append(countItems, MakeCountItem(ser, ts, size))
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
	log.W.Ln("GC needs to run")
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
	// don't think dedup is needed
	// 	var dupes []int
	// 	for i := range deleteItems {
	// 		if i == 0 {
	// 			continue
	// 		}
	// 		if bytes.Compare(deleteItems[i], deleteItems[i-1]) == 0 {
	// 			log.I.Ln("dupe")
	// 			dupes = append(dupes, i)
	// 		}
	// 	}
	// 	log.I.S(dupes)
	// 	tmp := make(DeleteItems, 0, len(deleteItems)-len(dupes))
	// skip:
	// 	for i := range deleteItems {
	// 		for j := range dupes {
	// 			if dupes[j] == i {
	// 				continue skip
	// 			}
	// 			tmp = append(tmp, deleteItems[i])
	// 		}
	// 	}
	// 	deleteItems = tmp
	log.I.Ln("found", lastIndex,
		"events to prune, which will bring current utilization down to",
		total-cumulative)
	return
}
