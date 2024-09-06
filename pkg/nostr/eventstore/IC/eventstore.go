package IC

import (
	"os"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/IConly"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/l2"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
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
	*l2.Backend
}

var _ eventstore.Store = (*Backend)(nil)

// GetBackend returns a l2.Backend that combines the two provided backends. It
// is assumed both were
func GetBackend(c context.T, wg *sync.WaitGroup, L1 *badger.Backend,
	L2 *IConly.Backend, pf time.Duration, po timestamp.T) (es eventstore.Store,
	signal event.C) {
	signal = make(event.C)
	es = &l2.Backend{
		Ctx:           c,
		WG:            wg,
		L1:            L1,
		L2:            L2,
		PollFrequency: pf,
		PollOverlap:   po,
		EventSignal:   signal,
	}
	return
}

// Init sets up the badger event store and connects to the configured IC
// canister.
//
// required params are address, canister ID and the badger event store size
// limit (which can be 0)
func (b *Backend) Init() (err error) {
	log.I.Ln("initializing badger/IC hybrid event store")
	return b.Backend.Init()
}

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
func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C,
	err error) {
	return b.Backend.QueryEvents(c, f)
}

// SaveEvent writes an event to the event store.
func (b *Backend) SaveEvent(c context.T, ev *event.T) (err error) {
	return b.Backend.SaveEvent(c, ev)
}
