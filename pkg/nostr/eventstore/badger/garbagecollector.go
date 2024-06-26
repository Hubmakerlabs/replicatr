package badger

import (
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/units"
)

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
		"GC check frequency %v %s",
		float32(b.DBSizeLimit)/units.Mb,
		float32(b.DBHighWater*b.DBSizeLimit/100)/units.Mb,
		float32(b.DBLowWater*b.DBSizeLimit/100)/units.Mb,
		units.Mb,
		b.GCFrequency,
		b.Path,
	)
	var err error
	if err = b.GCRun(); chk.E(err) {
	}
	GCticker := time.NewTicker(b.GCFrequency)
	// force sync to disk every so often, this might be normally about 10 minutes.
	syncTicker := time.NewTicker(b.GCFrequency * 10)
out:
	for {
		select {
		case <-b.Ctx.Done():
			log.W.Ln("stopping event GC ticker")
			break out
		case <-GCticker.C:
			// log.T.Ln("running GC", b.Path)
			if err = b.GCRun(); chk.E(err) {
			}
		case <-syncTicker.C:
			chk.E(b.DB.Sync())
		}
	}
	log.I.Ln("closing badger event store garbage collector")
}

func (b *Backend) GCRun() (err error) {
	log.T.Ln("running GC", b.Path)
	var pruneEvents, pruneIndexes DelItems
	if pruneEvents, pruneIndexes, err = b.GCMark(); chk.E(err) {
		return
	}
	if len(pruneEvents) < 1 && len(pruneIndexes) < 1 {
		log.I.Ln("GC sweep unnecessary")
		return
	}
	if err = b.GCSweep(pruneEvents, pruneIndexes); chk.E(err) {
		return
	}
	return
}
