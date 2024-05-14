package badger

import (
	"encoding/binary"
	"sort"
	"sync"
	"time"

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
	log.I.Ln("running event GC count")
	overallStart := time.Now()
	key := make([]byte, index.Len+serial.Len)
	prf := []byte{byte(index.Event)}
	_, counters := b.DB.EstimateSize([]byte{index.Counter.B()})
	log.I.F("estimate of number of events %d (%d bytes)", int(float64(counters)/22.398723239), counters)
	stream := b.DB.NewStream()
	stream.Prefix = prf
	var countMx sync.Mutex
	var totalCounter int
	stream.ChooseKey = func(item *badger.Item) (b bool) {
		item.KeyCopy(key)
		ser := serial.FromKey(key)
		// skip already pruned items
		size := uint32(item.ValueSize())
		totalCounter++
		if size == sha256.Size {
			return true
		}
		countMx.Lock()
		countItems = append(countItems, &count.Item{Serial: ser.Uint64(), Size: size})
		countMx.Unlock()
		return
	}
	started := time.Now()
	if err = stream.Orchestrate(b.Ctx); chk.E(err) {
		return
	}
	log.I.F("counted %d unpruned and %d pruned events in %v", len(countItems),
		time.Now().Sub(started))
	// second get the datestamps of the items
	stream = b.DB.NewStream()
	stream.Prefix = []byte{index.Counter.B()}
	v := make([]byte, createdat.Len)
	countFresh := make(count.Freshes, 0, totalCounter)
	stream.ChooseKey = func(item *badger.Item) (b bool) {
		item.KeyCopy(key)
		ser := serial.FromKey(key)
		s64 := ser.Uint64()
		countMx.Lock()
		countFresh = append(countFresh,
			&count.Fresh{
				Serial:    s64,
				Freshness: timestamp.FromUnix(int64(binary.BigEndian.Uint64(v))),
			})
		countMx.Unlock()
		return
	}
	if err = stream.Orchestrate(b.Ctx); chk.E(err) {
		return
	}
	countBySerial := count.ItemsBySerial(countItems)
	sort.Sort(countBySerial)
	sort.Sort(countFresh)
	// both slices are now sorted by serial, so we can now iterate the freshness
	// slice and write in the access timestamps to the countItems
	//
	// this provides the least amount of iteration and computation to essentially
	// zip two tables together
	var cursor int
	for i := range countFresh {
		if countFresh[i].Serial == countBySerial[cursor].Serial {
			countBySerial[cursor].Freshness = countFresh[i].Freshness
			// advance the serial
			cursor++
		}
	}
	total = countItems.Total()
	log.D.F("%d records; total size of data %0.6f MB %0.3f KB "+
		"high water %0.3f Mb computed in %v",
		len(countItems),
		float64(total)/units.Mb, float64(total)/units.Kb,
		float64(b.DBHighWater*b.DBSizeLimit/100)/units.Mb,
		time.Now().Sub(overallStart),
	)
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
	// sort.Sort(deleteItems)
	log.D.Ln("found", lastIndex,
		"events to prune, which will bring current utilization down to",
		total-cumulative)
	return
}
