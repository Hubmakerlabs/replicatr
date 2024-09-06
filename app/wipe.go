package app

import (
	bdb "github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
)

// Wipe clears the badgerDB local event store/cache.
func (rl *Relay) Wipe(store *bdb.Backend) (err error) {
	if err = store.Wipe(); chk.E(err) {
		return
	}
	return
}
