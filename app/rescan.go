package app

import bdb "github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"

// RescanAC clears and regenerates access counter records.
// todo: implement this
func (rl *Relay) RescanAC(store *bdb.Backend) (err error) {
	log.W.Ln("not implemented")
	return
}
