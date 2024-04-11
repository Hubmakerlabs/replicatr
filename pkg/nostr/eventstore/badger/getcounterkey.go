package badger

import (
	"mleku.dev/git/nostr/eventid"
	"mleku.dev/git/nostr/eventstore/badger/keys/id"
	"mleku.dev/git/nostr/eventstore/badger/keys/index"
	"mleku.dev/git/nostr/eventstore/badger/keys/serial"
)

// GetCounterKey returns the proper counter key for a given event ID.
func GetCounterKey(ev *eventid.T, ser []byte) (key []byte) {
	// evIDPrefix, err := hex.Dec((*ev)[:CounterIDLen*2].String())
	// if err != nil {
	// 	log.E.Ln("event was stored with bogus hex")
	// 	// this should never happen
	// 	panic(err)
	// }
	// key = make([]byte, 1+CounterIDLen)
	// key[0] = IndexCounterPrefix
	// copy(key[1:1+CounterIDLen], evIDPrefix)
	key = index.Counter.Key(id.New(*ev), serial.New(ser))
	log.I.F("counter key %0x %0x %0x", key[0], key[1:9], key[9:])
	return
}
