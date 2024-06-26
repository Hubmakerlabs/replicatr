package badger

import (
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/dgraph-io/badger/v4"
	"github.com/minio/sha256-simd"
)

func (b *Backend) GCSweep(evs, idxs DelItems) (err error) {
	// first we must gather all the indexes of the relevant events
	batch := b.DB.NewWriteBatch()
	defer batch.Cancel()
	started := time.Now()
	// var wg sync.WaitGroup
	// go func() {
	// 	wg.Add(1)
	// 	defer wg.Done()
	stream := b.DB.NewStream()
	// get all the event indexes to delete/prune
	stream.Prefix = []byte{index.Event.B()}
	stream.ChooseKey = func(item *badger.Item) (bo bool) {
		if item.KeySize() != 1+serial.Len {
			return
		}
		if item.IsDeletedOrExpired() {
			return
		}
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
		if b.HasL2 {
			// if it's already pruned, skip
			if item.ValueSize() == sha256.Size {
				return
			}
			// if there is L2 we are only pruning (replacing event with the ID hash)
			var evb []byte
			if evb, err = item.ValueCopy(nil); chk.E(err) {
				return
			}
			var ev *event.T
			if ev, err = nostrbinary.Unmarshal(evb); chk.E(err) {
				return
			}
			if err = batch.Set(key, ev.ID.Bytes()); chk.E(err) {
				return
			}
			return
		} else {
			// otherwise we are deleting
			if err = batch.Delete(key); chk.E(err) {
				return
			}
		}
		return
	}
	// execute the event prune/delete
	if err = stream.Orchestrate(b.Ctx); chk.E(err) {
		return
	}
	// }()
	// next delete all the indexes
	if len(idxs) > 0 && b.HasL2 {
		log.T.Ln("pruning indexes")
		// we have to remove everything
		prfs := [][]byte{{index.Event.B()}}
		prfs = append(prfs, index.FilterPrefixes...)
		prfs = append(prfs, []byte{index.Counter.B()})
		for _, prf := range prfs {
			stream := b.DB.NewStream()
			stream.Prefix = prf
			stream.ChooseKey = func(item *badger.Item) (bo bool) {
				if item.IsDeletedOrExpired() || item.KeySize() < serial.Len+1 {
					return
				}
				key := item.KeyCopy(nil)
				ser := serial.FromKey(key).Uint64()
				var found bool
				for _, idx := range idxs {
					if idx == ser {
						found = true
						break
					}
				}
				if !found {
					return
				}
				// log.I.F("deleting index %x %d", prf, ser)
				if err = batch.Delete(key); chk.E(err) {
					return
				}
				return
			}
			if err = stream.Orchestrate(b.Ctx); chk.E(err) {
				return
			}
			log.T.Ln("completed index prefix", prf)
		}
	}
	log.T.Ln("flushing batch")
	if err = batch.Flush(); chk.E(err) {
		return
	}
	if vlerr := b.DB.RunValueLogGC(0.5); vlerr == nil {
		log.I.Ln("value log cleaned up")
	}
	log.I.Ln("completed sweep in", time.Now().Sub(started), b.Path)
	return
}
