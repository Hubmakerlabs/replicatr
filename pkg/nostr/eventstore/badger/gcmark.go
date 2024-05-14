package badger

import (
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/count"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/units"
	"github.com/minio/sha256-simd"
)

type DelItems []uint64

// GCMark first gathers the serial, data size and last accessed information
// about all events and pruned events using GCCount then sorts the results of
// the events and indexes by least recently accessed and generates the set of
// serials of events that need to be deleted
func (b *Backend) GCMark() (pruneEvents, pruneIndexes DelItems, err error) {
	var unpruned, pruned count.Items
	var uTotal, pTotal int
	headroom := b.DBSizeLimit * (100 - b.DBHighWater) / 100
	lw, hw := headroom*b.DBLowWater/100, headroom*b.DBHighWater/100
	if unpruned, pruned, uTotal, pTotal, err = b.GCCount(); chk.E(err) {
		return
	}
	trigger := b.DBHighWater * b.DBSizeLimit / 100
	if uTotal > trigger {
		// run event GC mark
		sort.Sort(unpruned)
		pruneOff := uTotal - b.DBLowWater*b.DBSizeLimit/100
		var cumulative, lastIndex int
		for lastIndex = range unpruned {
			if cumulative > pruneOff {
				break
			}
			// if there is an L2 the ID and key remain
			if b.HasL2 {
				cumulative += int(unpruned[lastIndex].Size) - sha256.Size
			} else {
				cumulative += int(unpruned[lastIndex].Size) + serial.Len + 1
			}
			pruneEvents = append(pruneEvents, unpruned[lastIndex].Serial)
		}
		log.D.F("found %d events to prune, which will bring current "+
			"utilization down to %0.3f Gb %s",
			lastIndex, float64(uTotal-cumulative)/units.Gb, b.Path)
	}
	if b.HasL2 && pTotal < hw {
		// run index GC mark
		sort.Sort(pruned)
		var lastIndex int
		space := headroom
		// count the number of events until the low water mark
		for lastIndex = range pruned {
			if space < lw {
				break
			}
			space -= int(pruned[lastIndex].Size)
		}
		log.D.F("deleting %d indexes using %d bytes to bring pruned index size to %d",
			lastIndex+1, headroom-space, space)
		for i := range pruned {
			if i > lastIndex {
				break
			}
			pruneIndexes = append(pruneIndexes, pruned[i].Serial)
		}
	}
	return
}
