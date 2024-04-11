package badger

import (
	"encoding/hex"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/arb"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/kinder"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/pubkey"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

func getTagIndexPrefix(tagValue string) (key []byte, offset int) {
	// var key []byte // the key with full length for created_at and idx at the end, but not filled with these
	// var offset int // the offset -- i.e. where the prefix ends and the created_at and idx would start
	var err error
	if k, pkb, d := eventstore.GetAddrTagElements(tagValue); len(pkb) == 32 {
		// store value in the new special "a" tag index
		// k = make([]byte, 1+2+8+len(d)+4+4)
		// k[0] = indexTagAddrPrefix
		// binary.BigEndian.PutUint16(k[1:], kind)
		// copy(k[1+2:], pkb[0:8])
		// copy(k[1+2+8:], d)
		// offset = 1 + 2 + 8 + len(d)
		var pk *pubkey.T
		pk, _ = pubkey.NewFromBytes(pkb)
		els := []keys.Element{kinder.New(k), pk}
		if len(d) > 0 {
			els = append(els, arb.NewFromString(d))
		}
		key = index.TagAddr.Key(els...)

	} else if pkb, _ := hex.DecodeString(tagValue); len(pkb) == 32 {
		// store value as bytes
		// k = make([]byte, 1+8+4+4)
		// k[0] = indexTag32Prefix
		// copy(k[1:], vb[0:8])
		// offset = 1 + 8
		var pkk *pubkey.T
		if pkk, _ = pubkey.NewFromBytes(pkb); chk.E(err) {
			return
		}
		key = index.Tag32.Key(pkk)
	} else {
		// store whatever as utf-8
		// k = make([]byte, 1+len(tagValue)+4+4)
		// k[0] = indexTagPrefix
		// copy(k[1:], tagValue)
		// offset = 1 + len(tagValue)
		if len(tagValue) > 0 {
			var a *arb.T
			a = arb.NewFromString(tagValue)
			key = index.Tag.Key(a)
		}
		key = index.Tag.Key()
	}
	return key, offset
}

// func GetIndexKeysForEvent(evt *event.T, idx []byte) [][]byte {
// 	keys := make([][]byte, 0, 18)
//
// 	// indexes
// 	{
// 		// ~ by id
// 		idPrefix8, _ := hex.DecodeString(evt.ID[0 : 8*2].String())
// 		k := make([]byte, 1+8+4)
// 		k[0] = indexIdPrefix
// 		copy(k[1:], idPrefix8)
// 		copy(k[1+8:], idx)
// 		keys = append(keys, k)
// 	}
//
// 	{
// 		// ~ by pubkey+date
// 		pubkeyPrefix8, _ := hex.DecodeString(evt.PubKey[0 : 8*2])
// 		k := make([]byte, 1+8+4+4)
// 		k[0] = indexPubkeyPrefix
// 		copy(k[1:], pubkeyPrefix8)
// 		binary.BigEndian.PutUint32(k[1+8:], uint32(evt.CreatedAt))
// 		copy(k[1+8+4:], idx)
// 		keys = append(keys, k)
// 	}
//
// 	{
// 		// ~ by kind+date
// 		k := make([]byte, 1+2+4+4)
// 		k[0] = indexKindPrefix
// 		binary.BigEndian.PutUint16(k[1:], uint16(evt.Kind))
// 		binary.BigEndian.PutUint32(k[1+2:], uint32(evt.CreatedAt))
// 		copy(k[1+2+4:], idx)
// 		keys = append(keys, k)
// 	}
//
// 	{
// 		// ~ by pubkey+kind+date
// 		pubkeyPrefix8, _ := hex.DecodeString(evt.PubKey[0 : 8*2])
// 		k := make([]byte, 1+8+2+4+4)
// 		k[0] = indexPubkeyKindPrefix
// 		copy(k[1:], pubkeyPrefix8)
// 		binary.BigEndian.PutUint16(k[1+8:], uint16(evt.Kind))
// 		binary.BigEndian.PutUint32(k[1+8+2:], uint32(evt.CreatedAt))
// 		copy(k[1+8+2+4:], idx)
// 		keys = append(keys, k)
// 	}
//
// 	// ~ by tagvalue+date
// 	for i, tt := range evt.Tags {
// 		if len(tt) < 2 || len(tt[0]) != 1 || len(tt[1]) == 0 || len(tt[1]) > 100 {
// 			// not indexable
// 			continue
// 		}
// 		firstIndex := slices.IndexFunc(evt.Tags, func(t tag.T) bool { return len(t) >= 2 && t[1] == tt[1] })
// 		if firstIndex != i {
// 			// duplicate
// 			continue
// 		}
//
// 		// get key prefix (with full length) and offset where to write the last parts
// 		k, offset := getTagIndexPrefix(tt[1])
//
// 		// write the last parts (created_at and idx)
// 		binary.BigEndian.PutUint32(k[offset:], uint32(evt.CreatedAt))
// 		copy(k[offset+4:], idx)
// 		keys = append(keys, k)
// 	}
//
// 	{
// 		// ~ by date only
// 		k := make([]byte, 1+4+4)
// 		k[0] = indexCreatedAtPrefix
// 		binary.BigEndian.PutUint32(k[1:], uint32(evt.CreatedAt))
// 		copy(k[1+4:], idx)
// 		keys = append(keys, k)
// 	}
//
// 	return keys
// }
