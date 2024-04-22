package badgerbadger

import (
	"os"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/l2"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

// Backend is a hybrid badger/badger eventstore where L1 will have GC enabled
// and L2 will not. This is mainly for testing, as both are local.
type Backend struct {
	*l2.Backend
}

var _ eventstore.Store = (*Backend)(nil)

// GetBackend returns a l2.Backend that combines two differently configured
// backends... the settings need to be configured in the badger.Backend data
// structure before calling this.
func GetBackend(c context.T, wg *sync.WaitGroup, L1 *badger.Backend,
	L2 *badger.Backend) (es eventstore.Store) {
	es = &l2.Backend{Ctx: c, WG: wg, L1: L1, L2: L2}
	return
}

// Init sets up the badger event store and connects to the configured IC
// canister.
//
// required params are address, canister ID and the badger event store size
// limit (which can be 0)
func (b *Backend) Init() (err error) { return b.Backend.Init() }

// Close the connection to the database.
// IC is a request/response API authing at each request.
func (b *Backend) Close() { b.Backend.Close() }

// CountEvents returns the number of events found matching the filter.
func (b *Backend) CountEvents(c context.T, f *filter.T) (count int, err error) {
	return b.Backend.CountEvents(c, f)
}

// DeleteEvent removes an event from the event store.
func (b *Backend) DeleteEvent(c context.T, ev *event.T) (err error) {
	return b.Backend.DeleteEvent(c, ev)
}

// QueryEvents searches for events that match a filter and returns them
// asynchronously over a provided channel.
func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C, err error) {
	return b.Backend.QueryEvents(c, f)
}

// SaveEvent writes an event to the event store.
func (b *Backend) SaveEvent(c context.T, ev *event.T) (err error) {
	return b.Backend.SaveEvent(c, ev)
}
