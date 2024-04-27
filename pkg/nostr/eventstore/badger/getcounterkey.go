package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
)

// GetCounterKey returns the proper counter key for a given event ID.
func GetCounterKey(ser []byte) (key []byte) {
	key = index.Counter.Key(serial.New(ser))
	log.T.F("counter key %0x %0x", key[0], key[1:9])
	return
}
