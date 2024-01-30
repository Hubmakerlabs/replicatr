package replicatr

import "github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"

// Import a collection of JSON events from stdin or from one or more files, line
// structured JSON.
func (rl *Relay) Import(db *badger.Backend, filename []string) {
	rl.D.Ln("running export subcommand")

}
