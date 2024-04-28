package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
)

// GetCounterKey returns the proper counter key for a given event ID.
func GetCounterKey(ser []byte) (key []byte) {
	seri := serial.New(ser)
	key = index.Counter.Key(seri)
	log.T.F("counter key %d", seri.Uint64())
	return
}
