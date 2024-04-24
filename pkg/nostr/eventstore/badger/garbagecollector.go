package badger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/del"
	"github.com/Hubmakerlabs/replicatr/pkg/units"
)

// GarbageCollector starts up a ticker that runs a check on space utilisation
// and when it exceeds the high-water mark, prunes back to the low-water mark.
//
// This function should be invoked as a goroutine, and will terminate when the
// backend context is canceled.
func (b *Backend) GarbageCollector() {
	log.W.Ln("starting badger back-end garbage collector:")
	log.I.F("max size %0.3f MB; "+
		"high water %0.3f MB; "+
		"low water %0.3f MB "+
		"(MB = %d bytes) "+
		"GC check frequency %v",
		float32(b.DBSizeLimit)/units.Mb,
		float32(b.DBHighWater*b.DBSizeLimit/100)/units.Mb,
		float32(b.DBLowWater*b.DBSizeLimit/100)/units.Mb,
		units.Mb,
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
			log.T.Ln("running GC check")
			if err = b.GCRun(); chk.E(err) {

			}
		}
	}
	log.I.Ln("closing badger event store garbage collector")
}

func (b *Backend) GCRun() (err error) {
	log.T.Ln("running garbage collector check")
	var deleteItems del.Items
	if deleteItems, err = b.GCCount(); chk.E(err) {
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
	if err = b.Prune(deleteItems); chk.E(err) {
		return
	}
	return
}
