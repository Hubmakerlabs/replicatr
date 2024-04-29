package IConly

import (
	"os"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/ic/agent"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayinfo"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

// Backend is a pure Internet Computer Protocol based event store. All queries
// are performed to a remote data store.
type Backend struct {
	Ctx             context.T
	WG              *sync.WaitGroup
	IC              *agent.Backend
	Inf             *relayinfo.T
	CanisterAddr    string
	CanisterId      string
	PrivateCanister bool
	SecKey          string
}

var _ eventstore.Store = (*Backend)(nil)

// Init  connects to the configured IC canister.
func (b *Backend) Init() (err error) {
	if b.CanisterAddr == "" || b.CanisterId == "" {
		return log.E.Err("missing required canister parameters, got addr: \"%s\" and id: \"%s\"",
			b.CanisterAddr, b.CanisterId)
	}
	if b.IC, err = agent.New(b.Ctx, b.CanisterId, b.CanisterAddr, b.SecKey); chk.E(err) {
		return
	}
	return
}

// Close the connection to the database. This is a no-op because the queries are
// stateless.
func (b *Backend) Close() {}

// CountEvents returns the number of events found matching the filter. This is
// synchronous.
func (b *Backend) CountEvents(c context.T, f *filter.T) (count int, err error) {
	return b.IC.CountEvents(c, f)
}

// DeleteEvent removes an event from the event store. This is synchronous.
func (b *Backend) DeleteEvent(c context.T, ev *event.T) (err error) {
	return b.IC.DeleteEvent(c, ev)
}

// QueryEvents searches for events that match a filter and returns them
// asynchronously over a provided channel.
//
// This is asynchronous, it will never return an error.
func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C, err error) {
	ch = make(event.C)
	go func() {
		log.D.Ln("querying IC with filter", f.ToObject().String())
		if ch, err = b.IC.QueryEvents(c, f); chk.E(err) {
		}
	}()
	return
}

// SaveEvent writes an event to the event store. This is synchronous.
func (b *Backend) SaveEvent(c context.T, ev *event.T) (err error) {
	log.I.Ln("saving event to IC", ev.ToObject().String())
	if err = b.IC.SaveEvent(c, ev); chk.E(err) {
		return
	}
	return
}
