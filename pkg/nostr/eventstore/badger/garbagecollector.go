package badger

import (
	"bytes"
	"encoding/binary"
	"sort"
	"time"

	"github.com/dgraph-io/badger/v4"
	"mleku.dev/git/nostr/eventstore/badger/keys/index"
	"mleku.dev/git/nostr/timestamp"
)

func (b *Backend) GarbageCollector() {
	log.T.Ln("starting badger back-end garbage collector:")
	log.I.F("max size %0.3f MB; "+
		"high water %0.3f MB; "+
		"low water %0.3f MB "+
		"(MB = %d bytes) "+
		"GC check frequency %v",
		float32(b.DBSizeLimit)/Megabyte,
		float32(b.DBHighWater)/Megabyte,
		float32(b.DBLowWater)/Megabyte,
		Megabyte,
		b.GCFrequency,
	)
	var err error
	if err = b.GCRun(); chk.E(err) {
	}
	gcTicker := time.NewTicker(b.GCFrequency)
out:
	for {
		select {
		case <-b.Ctx.Done():
			log.W.Ln("backend context done")
			break out
		case <-gcTicker.C:
			log.I.Ln("running GC check")
			if err = b.GCRun(); chk.E(err) {

			}
		}
	}
	log.I.Ln("closing badger event store garbage collector")
}

var counterPrefix = []byte{byte(index.Counter)}

type CountItem struct {
	Key       []byte
	Value     []byte
	Size      uint32
	Freshness timestamp.T
}

type CountItems []CountItem

func (c CountItems) Total() (total int) {
	for i := range c {
		total += int(c[i].Size)
	}
	return
}

func (c CountItems) Len() int           { return len(c) }
func (c CountItems) Less(i, j int) bool { return c[i].Freshness < c[j].Freshness }
func (c CountItems) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

func (b *Backend) GCRun() (err error) {
	log.T.Ln("running garbage collector check")
	// calculate current size
	var countItems CountItems
	var deleteItems, deleteValues [][]byte
	err = b.View(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.IteratorOptions{
			Prefix: counterPrefix,
		})
		for it.Rewind(); it.ValidForPrefix(counterPrefix); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			var v []byte
			if v, err = item.ValueCopy(nil); chk.E(err) || len(v) == 0 {
				continue
			}
			ci := CountItem{
				Key:       key,
				Value:     v,
				Freshness: timestamp.T(binary.BigEndian.Uint64(v[:8])),
				Size:      binary.BigEndian.Uint32(v[8:]),
			}
			countItems = append(countItems, ci)
		}
		it.Close()
		total := countItems.Total()
		log.I.F("total size of data %0.6f MB %0.3f KB high water %0.3f Mb",
			float64(total)/Megabyte, float64(total)/1000,
			float64(b.DBHighWater)/Megabyte)
		if total < b.DBHighWater {
			return
		}
		log.T.Ln("GC needs to run")
		sort.Sort(countItems)
		pruneOff := total - b.DBLowWater
		log.T.Ln("will delete nearest to", pruneOff,
			"bytes of events from the event store from the most stale")
		var cumulative, lastIndex int
		for lastIndex = range countItems {
			if cumulative > pruneOff {
				break
			}
			cumulative += int(countItems[lastIndex].Size)
			deleteItems = append(deleteItems, countItems[lastIndex].Key)
			deleteValues = append(deleteValues, countItems[lastIndex].Value)
		}
		log.T.Ln("found", lastIndex,
			"events to prune, which will bring current utilization down to",
			total-cumulative)
		return
	})
	// toDelete := len(deleteItems)
	var clearKeys [][]byte
	err = b.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().Key()
			if k[0] != byte(index.Id) {
				continue
			}
			for i := range deleteItems {
				if bytes.Compare(k[1:9], deleteItems[i][1:9]) != 0 {
					continue
				}
				log.I.Ln("adding key", k)
				clearKeys = append(clearKeys, k)

			}
		}
		it.Close()
		return nil
	})
	// now we have all the keys of the events we want to clear, and all of the
	// event IDs that we can use to find the access counter entries and clear
	// the "storage utilisation" field of so that the event can be excluded from
	// future calls to GCRun.
	// log.T.S(clearKeys)
	err = b.Update(func(txn *badger.Txn) (err error) {
		for i := range clearKeys {
			if err = txn.Set(clearKeys[i], []byte{}); !chk.E(err) {
				log.I.Ln("cleared index", clearKeys[i])
			}
		}
		for i := range deleteItems {
			// set size of access counter record to zero to match the record
			copy(deleteValues[i][8:], make([]byte, 4))
			if err = txn.Set(deleteItems[i], deleteValues[i]); !chk.E(err) {
				log.I.Ln("cleared access counter size", deleteItems[i])
			}

		}
		return
	})
	return
}
