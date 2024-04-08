package IC

import (
	"os"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/ic/agent"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/eventstore"
	"mleku.dev/git/nostr/eventstore/badger"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/kinds"
	"mleku.dev/git/nostr/relayinfo"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Backend struct {
	// Badger backend must be populated
	Badger *badger.Backend
	IC     *agent.Backend
	Ctx    context.T
	WG     *sync.WaitGroup
	Inf    *relayinfo.T
	Params []string
}

var _ eventstore.Store = (*Backend)(nil)

// Init sets up the badger event store and connects to the configured IC
// canister.
//
// required params are address, canister ID and the badger event store size
// limit (which can be 0)
func (b *Backend) Init() (err error) {

	if len(b.Params) < 3 {
		return log.E.Err("not enough parameters for IC event store Init, "+
			"got %d, require %d: %v", len(b.Params), 3, b.Params)
	}
	addr, canisterId := b.Params[0], b.Params[1]
	if err = b.Badger.Init(); chk.D(err) {
		return
	}
	if b.IC, err = agent.New(b.Ctx, canisterId, addr); chk.E(err) {
		return
	}
	return
}

// Close the connection to the database.
func (b *Backend) Close() { b.Badger.Close() }

// Serial returns the serial code for the database.
func (b *Backend) Serial() []byte {
	by := b.Badger.Serial()
	return by
}

// CountEvents returns the number of events found matching the filter.
func (b *Backend) CountEvents(c context.T, f *filter.T) (count int, err error) {
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
	if count, err = b.Badger.CountEvents(c, f); chk.E(err) {
		return
	}
	var bcounter int
	if bcounter, err = b.Badger.CountEvents(c, icFilter); chk.E(err) {
		return
	}
	var counter int
	counter, err = b.IC.CountEvents(c, icFilter)
	// add the result of the bigger of local vs ic so the effective count is
	// closer to correct
	if counter > bcounter {
		count += counter
	} else {
		count += bcounter
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
