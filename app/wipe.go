package app

import (
	bdb "mleku.dev/git/nostr/eventstore/badger"
)

// Wipe clears the badgerDB local event store/cache.
func (rl *Relay) Wipe(store *bdb.Backend) (err error) {
	chk.E(store.DB.DropPrefix([][]byte{
		{bdb.RawEventStorePrefix},
		{bdb.IndexCreatedAtPrefix},
		{bdb.IndexIdPrefix},
		{bdb.IndexKindPrefix},
		{bdb.IndexPubKeyPrefix},
		{bdb.IndexPubKeyKindPrefix},
		{bdb.IndexTagPrefix},
		{bdb.IndexTag32Prefix},
		{bdb.IndexTagAddrPrefix},
		{bdb.IndexCounterPrefix},
	}...))
	if err = store.DB.RunValueLogGC(0.8); chk.E(err) {
	}
	chk.E(store.DB.Close())
	return
}
