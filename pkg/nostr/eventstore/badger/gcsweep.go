package badger

import (
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/dgraph-io/badger/v4"
)

func (b *Backend) GCSweep(evs, idxs DelItems) (err error) {
	// first we must gather all the indexes of the relevant events
	var countMx sync.Mutex
	// evIndexes are the indexes that are to be deleted or if L2 pruned
	var evIndexes, deleteIndexes [][]byte
	pruneStarted := time.Now()
	stream := b.DB.NewStream()
	var wg sync.WaitGroup
	// get all the event indexes to delete/prune
	stream.Prefix = []byte{index.Event.B()}
	stream.ChooseKey = func(item *badger.Item) (bo bool) {
		key := item.KeyCopy(nil)
		ser := serial.FromKey(key).Uint64()
		var found bool
		for i := range evs {
			if evs[i] == ser {
				found = true
				break
			}
		}
		if !found {
			return
		}
		countMx.Lock()
		if b.HasL2 {
			// if there is L2 we are only pruning
			evIndexes = append(evIndexes, key)
		} else {
			// otherwise we are deleting
			deleteIndexes = append(deleteIndexes, key)
		}
		countMx.Unlock()

		return
	}
	// grab all the events
	go func() {
		wg.Add(1)
		if err = stream.Orchestrate(b.Ctx); chk.E(err) {
			return
		}
		wg.Done()
	}()
	// next we need to scan all the indexes and pick the ones matching the
	if len(idxs) > 0 && b.HasL2 {
		for _, prf := range index.FilterPrefixes {
			stream.Prefix = prf
			stream.ChooseKey = func(item *badger.Item) (bo bool) {
				key := item.KeyCopy(nil)
				countMx.Lock()
				deleteIndexes = append(deleteIndexes, key)
				countMx.Unlock()
				return
			}
			go func(prf []byte) {
				wg.Add(1)
				if err = stream.Orchestrate(b.Ctx); chk.E(err) {
					return
				}
				wg.Done()
			}(prf)
		}

	}
	// wb := b.DB.NewWriteBatch()
	_ = pruneStarted
	return
}
