package badger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/del"
	"github.com/Hubmakerlabs/replicatr/pkg/units"
)

const IndexGCSkip = 5

// GarbageCollector starts up a ticker that runs a check on space utilisation
// and when it exceeds the high-water mark, prunes back to the low-water mark.
//
// This function should be invoked as a goroutine, and will terminate when the
// backend context is canceled.
func (b *Backend) GarbageCollector() {
	log.D.F("starting badger back-end garbage collector: max size %0.3f MB; "+
		"high water %0.3f MB; "+
		"low water %0.3f MB "+
		"(MB = %d bytes) "+
		"event GC check frequency %v, index GC frequency %v",
		float32(b.DBSizeLimit)/units.Mb,
		float32(b.DBHighWater*b.DBSizeLimit/100)/units.Mb,
		float32(b.DBLowWater*b.DBSizeLimit/100)/units.Mb,
		units.Mb,
		b.GCFrequency, b.GCFrequency*IndexGCSkip,
	)
	var err error
	if err = b.EventGCRun(); chk.E(err) {
	}
	if b.HasL2 {
		indexGCticker := time.NewTicker(b.GCFrequency * IndexGCSkip)
		go func() {
		out:
			for {
				select {
				case <-b.Ctx.Done():
					log.W.Ln("stopping index GC ticker")
					break out
				case <-indexGCticker.C:
					log.T.Ln("running index GC")
					if err = b.IndexGCRun(); chk.E(err) {
					}
				}
			}

		}()
	}
	eventGCticker := time.NewTicker(b.GCFrequency)
	// force sync to disk every so often, this might be normally about 10 minutes.
	syncTicker := time.NewTicker(b.GCFrequency * 10)
out:
	for {
		select {
		case <-b.Ctx.Done():
			log.W.Ln("stopping event GC ticker")
			break out
		case <-eventGCticker.C:
			log.T.Ln("running event GC")
			if err = b.EventGCRun(); chk.E(err) {
			}
		case <-syncTicker.C:
			chk.E(b.DB.Sync())
		}
	}
	log.I.Ln("closing badger event store garbage collector")
}

func (b *Backend) EventGCRun() (err error) {
	var deleteItems del.Items
	if deleteItems, err = b.EventGCMark(); chk.E(err) {
		return
	}
	if len(deleteItems) < 1 {
		return
	}
	var delList string
	for i := range deleteItems {
		if i != 0 {
			delList += ", "
		}
		delList += fmt.Sprint(binary.BigEndian.Uint64(deleteItems[i]))
	}
	// log.I.Ln("pruning:", delList)
	if err = b.EventGCSweep(deleteItems); chk.E(err) {
		return
	}
	// b.EventGCCount()
	return
}

func (b *Backend) IndexGCRun() (err error) {
	var toDelete []uint64
	if toDelete, err = b.IndexGCMark(); chk.E(err) {
		return
	}
	if err = b.IndexGCSweep(toDelete); chk.E(err) {
		return
	}
	return
}
