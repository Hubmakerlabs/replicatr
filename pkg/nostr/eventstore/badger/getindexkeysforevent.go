package badger

import (
	"golang.org/x/exp/slices"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/eventstore/badger/keys"
	"mleku.dev/git/nostr/eventstore/badger/keys/arb"
	"mleku.dev/git/nostr/eventstore/badger/keys/createdat"
	"mleku.dev/git/nostr/eventstore/badger/keys/id"
	"mleku.dev/git/nostr/eventstore/badger/keys/index"
	"mleku.dev/git/nostr/eventstore/badger/keys/kinder"
	"mleku.dev/git/nostr/eventstore/badger/keys/pubkey"
	"mleku.dev/git/nostr/eventstore/badger/keys/serial"
	"mleku.dev/git/nostr/tag"
)

// GetIndexKeysForEvent generates all the index keys required to filter for
// events. evtSerial should be the output of Serial() which gets a unique,
// monotonic counter value for each new event.
func GetIndexKeysForEvent(ev *event.T, evtSerial []byte) (keyz [][]byte) {

	var err error
	keyz = make([][]byte, 0, 18)
	// be := binary.BigEndian
	ID := id.New(ev.ID)
	ser := serial.New(evtSerial)
	CA := createdat.New(ev.CreatedAt)
	K := kinder.New(ev.Kind)
	PK, _ := pubkey.New(ev.PubKey)
	// indexes
	{ // ~ by id
		// idPrefix8, _ := hex.Dec(ev.ID.String()[0 : 8*2])
		// k := make([]byte, 1+8+SerialLen)
		// k[0] = Id
		// copy(k[1:], idPrefix8)
		// copy(k[1+8:], evtSerial)
		k := index.Id.Key(ID, ser)
		log.T.F("id key: %x %0x %0x", k[0], k[1:9], k[9:])
		keyz = append(keyz, k)
	}
	{ // ~ by pubkey+date
		// pubkeyPrefix8, _ := hex.Dec(ev.PubKey[0 : 8*2])
		// k := make([]byte, 1+8+8+SerialLen)
		// k[0] = prefixes.Pubkey
		// copy(k[1:], pubkeyPrefix8)
		// be.PutUint32(k[1+8:], uint32(ev.CreatedAt))
		// copy(k[1+8+8:], evtSerial)
		// keyz = append(keyz, k)
		k := index.Pubkey.Key(PK, CA, ser)
		log.T.F("pubkey + date key: %x %0x %0x %0x",
			k[0], k[1:9], k[9:17], k[17:])
		keyz = append(keyz, k)
	}
	{ // ~ by kind+date
		// k := make([]byte, 1+2+4+SerialLen)
		// k[0] = prefixes.Kind
		// be.PutUint16(k[1:], uint16(ev.Kind))
		// be.PutUint32(k[1+2:], uint32(ev.CreatedAt))
		// copy(k[1+2+4:], evtSerial)
		// keyz = append(keyz, k)
		k := index.Kind.Key(K, CA, ser)
		log.T.F("kind + date key: %x %0x %0x %0x",
			k[0], k[1:3], k[3:11], k[11:])
		keyz = append(keyz, k)
	}
	{ // ~ by pubkey+kind+date
		// pubkeyPrefix8, _ := hex.Dec(ev.PubKey[0 : 8*2])
		// k := make([]byte, 1+8+2+4+SerialLen)
		// k[0] = prefixes.PubkeyKind
		// copy(k[1:], pubkeyPrefix8)
		// be.PutUint16(k[1+8:], uint16(ev.Kind))
		// be.PutUint32(k[1+8+2:], uint32(ev.CreatedAt))
		// copy(k[1+8+2+4:], evtSerial)
		// keyz = append(keyz, k)
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
		// // write the last parts (created_at and evtSerial)
		// be.PutUint32(k[offset:], uint32(ev.CreatedAt))
		// copy(k[offset+4:], evtSerial)
		var tp []byte
		if tp, err = GetTagKeyPrefix(t[1]); chk.E(err) {
			return
		}
		k := keys.Write(arb.New(tp), CA, ser)
		log.T.F("tag '%s': '%s' key %x", t[0], t[1], k)
		keyz = append(keyz, k)
	}
	{ // ~ by date only
		// k := make([]byte, 1+4+SerialLen)
		// k[0] = prefixes.CreatedAt
		// be.PutUint32(k[1:], uint32(ev.CreatedAt))
		// copy(k[1+4:], evtSerial)
		// keyz = append(keyz, k)
		k := index.CreatedAt.Key(CA, ser)
		log.T.F("date key: %x %0x %0x", k[0], k[1:9], k[9:])
		keyz = append(keyz, k)
	}
	return
}
