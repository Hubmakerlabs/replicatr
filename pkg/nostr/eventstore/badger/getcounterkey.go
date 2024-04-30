package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
)

// GetCounterKey returns the proper counter key for a given event ID.
func GetCounterKey(ser *serial.T) (key []byte) {
	key = index.Counter.Key(ser)
	// log.T.F("counter key %d %d", index.Counter, ser.Uint64())
	return
}
