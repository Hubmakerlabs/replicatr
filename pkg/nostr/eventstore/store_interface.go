package eventstore

import (
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
)

// Store is a persistence layer for nostr events handled by a relay.
type Store interface {
	// Init is called at the very beginning by [Server.Start], after
	// [Relay.Init], allowing a storage to initialize its internal resources.
	// The parameters can be used by the database implementations to set custom
	// parameters such as cache management and other relevant parameters to the
	// specific implementation.
	Init() (err error)
	// Close must be called after you're done using the store, to free up
	// resources and so on.
	Close()
	// QueryEvents is invoked upon a client's REQ as described in NIP-01. it
	// should return a channel with the events as they're recovered from a
	// database. the channel should be closed after the events are all
	// delivered.
	QueryEvents(c context.T, f *filter.T) (ch event.C, err error)
	// CountEvents performs the same work as QueryEvents but instead of
	// delivering the events that were found it just returns the count of events
	CountEvents(c context.T, f *filter.T) (count int, err error)
	// DeleteEvent is used to handle deletion events, as per NIP-09.
	DeleteEvent(c context.T, ev *event.T) (err error)
	// SaveEvent is called once Relay.AcceptEvent reports true.
	SaveEvent(c context.T, ev *event.T) (err error)
}
