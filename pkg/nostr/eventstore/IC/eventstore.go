package IC

import (
	"os"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	bdb "github.com/dgraph-io/badger/v4"

	"github.com/Hubmakerlabs/replicatr/pkg/ic/agent"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/del"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayinfo"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

// Backend is a hybrid badger/Internet Computer based event store.
//
// All search indexes are retained even if event data is pruned to reduce
// storage space for the relay, when events are pruned, their data is replaced
// with the event ID to signify they must be fetched and restored to the raw
// event key.
//
// An alternative data store could be created that purely relies on the IC, or
// alternatively prunes search indexes to minimize storage used by pruned
// events, but these would be slower at retrieval and require a more complex
// cache algorithm.
type Backend struct {
	// Badger backend must be populated
	Badger          *badger.Backend
	IC              *agent.Backend
	Ctx             context.T
	WG              *sync.WaitGroup
	Inf             *relayinfo.T
	CanisterAddr    string
	CanisterId      string
	PrivateCanister bool
}

var _ eventstore.Store = (*Backend)(nil)

// ICDelete manages the specific way that records are deleted for the IC data
// store.
func (b *Backend) ICDelete(serials del.Items) (err error) {
	err = b.Badger.Update(func(txn *bdb.Txn) (err error) {
		it := txn.NewIterator(bdb.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().Key()
			// check if key matches any of the serials
			for i := range serials {
				if serial.Match(k, serials[i]) {
					// decode the event, store the ID only in place of where the event was.
					var v []byte
					if v, err = it.Item().ValueCopy(nil); chk.E(err) {
						continue
					}
					var evt *event.T
					if evt, err = nostrbinary.Unmarshal(v); chk.E(err) {
						continue
					}
					if err = txn.Set(k, evt.ID.Bytes()); chk.E(err) {
						continue
					}
					break
				}
			}
		}
		return
	})
	chk.E(err)
	log.T.Ln("completed prune")
	chk.E(b.Badger.DB.Sync())
	return
}

// Init sets up the badger event store and connects to the configured IC
// canister.
//
// required params are address, canister ID and the badger event store size
// limit (which can be 0)
func (b *Backend) Init() (err error) {
	if b.CanisterAddr == "" || b.CanisterId == "" {
		return log.E.Err("missing required canister parameters, got addr: \"%s\" and id: \"%s\"",
			b.CanisterAddr, b.CanisterId)
	}
	if err = b.Badger.Init(); chk.D(err) {
		return
	}
	// start up the badger GC
	go b.Badger.GarbageCollector()
	if b.IC, err = agent.New(b.Ctx, b.CanisterId, b.CanisterAddr); chk.E(err) {
		return
	}
	return
}

// Close the connection to the database.
// IC is a request/response API authing at each request.
func (b *Backend) Close() {
	b.Badger.Close()
}

// CountEvents returns the number of events found matching the filter.
func (b *Backend) CountEvents(c context.T, f *filter.T) (count int, err error) {
	if count, err = b.Badger.CountEvents(c, f); chk.E(err) {
		return
	}
	return
}

// DeleteEvent removes an event from the event store.
func (b *Backend) DeleteEvent(c context.T, ev *event.T) (err error) {
	if kinds.IsPrivileged(ev.Kind) {
		log.D.Ln("deleting privileged event in relay store")
		return b.Badger.DeleteEvent(c, ev)
	}
	log.D.Ln("deleting event on relay store")
	if err = b.Badger.DeleteEvent(c, ev); chk.E(err) {
	}
	log.D.Ln("deleting event on IC")
	return b.IC.DeleteEvent(c, ev)
}

// QueryEvents searches for events that match a filter and returns them
// asynchronously over a provided channel.
func (b *Backend) QueryEvents(c context.T, f *filter.T) (C event.C, err error) {
	C = make(event.C)
	var forBadger, forIC kinds.T
	for i := range f.Kinds {
		if kinds.IsPrivileged(f.Kinds[i]) {
			forBadger = append(forBadger, f.Kinds[i])
		} else {
			forIC = append(forIC, f.Kinds[i])
		}
	}
	icFilter := f.Duplicate()
	f.Kinds = forBadger
	icFilter.Kinds = forIC
	if len(forBadger) > 0 {
		log.D.Ln("querying relay store with filter", f.ToObject().String())
		if C, err = b.Badger.QueryEvents(c, f); chk.E(err) {
		}
	}
	if len(forIC) > 0 {
		// todo: merge results of these two to reduce bandwidth
		log.D.Ln("querying relay store for events first")
		if C, err = b.Badger.QueryEvents(c, icFilter); chk.E(err) {
		}
		log.D.Ln("querying IC with filter", icFilter.ToObject().String())
		if C, err = b.IC.QueryEvents(c, f); chk.E(err) {
		}
	}
	return
}

// SaveEvent writes an event to the event store.
func (b *Backend) SaveEvent(c context.T, ev *event.T) (err error) {
	if kinds.IsPrivileged(ev.Kind) {
		log.I.Ln("saving privileged event to relay store",
			ev.ToObject().String())
		return b.Badger.SaveEvent(c, ev)
	}
	log.I.Ln("saving event to relay store first",
		ev.ToObject().String())
	if err = b.Badger.SaveEvent(c, ev); chk.E(err) {
	}
	log.I.Ln("saving event to IC", ev.ToObject().String())
	go func() {
		if err := b.IC.SaveEvent(c, ev); chk.E(err) {
			// not really much we can do on this end if that end fails, and this
			// slows down the return of the ok envelope greatly.
			//
			// since we know it's in the badger cache perhaps we can retry if
			// it is a transient error, something for later
			// todo maybe change to an error channel and async...
		}
	}()
	return
}
