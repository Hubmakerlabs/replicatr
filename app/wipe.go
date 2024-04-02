package app

import (
	bdb "mleku.dev/git/nostr/eventstore/badger"
)

// Wipe clears the badgerDB local event store/cache.
func (rl *Relay) Wipe(store *bdb.Backend) (err error) {
	chk.E(store.DB.DropPrefix([][]byte{
		{0},
		{1},
		{2},
		{3},
		{4},
		{5},
		{6},
		{7},
		{8},
		{9},
	}...))
	if err = store.DB.RunValueLogGC(0.8); chk.E(err) {
	}
	chk.E(store.DB.Close())
	return
}
