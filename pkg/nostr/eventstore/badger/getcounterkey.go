package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
)

// GetCounterKey returns the proper counter key for a given event ID.
func GetCounterKey(evID *eventid.T, ser []byte) (key []byte) {
	key = index.Counter.Key(id.New(*evID), serial.New(ser))
	log.T.F("counter key %0x %0x %0x", key[0], key[1:9], key[9:])
	return
}
