package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/kinder"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/pubkey"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"golang.org/x/exp/slices"
)

// GetIndexKeysForEvent generates all the index keys required to filter for
// events. evtSerial should be the output of Serial() which gets a unique,
// monotonic counter value for each new event.
func GetIndexKeysForEvent(ev *event.T, ser *serial.T) (keyz [][]byte) {

	var err error
	keyz = make([][]byte, 0, 18)
	ID := id.New(ev.ID)
	CA := createdat.New(ev.CreatedAt)
	K := kinder.New(ev.Kind)
	PK, _ := pubkey.New(ev.PubKey)
	// indexes
	{ // ~ by id
		k := index.Id.Key(ID, ser)
		log.T.F("id key: %x %0x %0x", k[0], k[1:9], k[9:])
		keyz = append(keyz, k)
	}
	{ // ~ by pubkey+date
		k := index.Pubkey.Key(PK, CA, ser)
		log.T.F("pubkey + date key: %x %0x %0x %0x",
			k[0], k[1:9], k[9:17], k[17:])
		keyz = append(keyz, k)
	}
	{ // ~ by kind+date
		k := index.Kind.Key(K, CA, ser)
		log.T.F("kind + date key: %x %0x %0x %0x",
			k[0], k[1:3], k[3:11], k[11:])
		keyz = append(keyz, k)
	}
	{ // ~ by pubkey+kind+date
		k := index.PubkeyKind.Key(PK, K, CA, ser)
		log.T.F("pubkey + kind + date key: %x %0x %0x %0x %0x",
			k[0], k[1:9], k[9:11], k[11:19], k[19:])
		keyz = append(keyz, k)
	}
	// ~ by tag value + date
	for i, t := range ev.Tags {
		if len(t) < 2 || // there is no value field
			// the tag is not a-zA-Z probably (this would permit arbitrary other
			// single byte chars)
			len(t[0]) != 1 ||
			// the second field is zero length
			len(t[1]) == 0 ||
			// the second field is more than 100 characters long
			len(t[1]) > 100 {
			// any of the above is true then the tag is not indexable
			continue
		}
		firstIndex := slices.IndexFunc(ev.Tags,
			func(ti tag.T) bool {
				return len(t) >= 2 && ti[1] == t[1]
			})
		if firstIndex != i {
			// duplicate
			continue
		}
		// get key prefix (with full length) and offset where to write the last
		// parts
		prf, elems := index.P(0), []keys.Element(nil)
		if prf, elems, err = GetTagKeyElements(t[1], CA, ser); chk.E(err) {
			return
		}
		k := prf.Key(elems...)
		log.T.F("tag '%s': %v key %x", t[0], t[1:], k)
		keyz = append(keyz, k)
	}
	{ // ~ by date only
		k := index.CreatedAt.Key(CA, ser)
		log.T.F("date key: %x %0x %0x", k[0], k[1:9], k[9:])
		keyz = append(keyz, k)
	}
	return
}
