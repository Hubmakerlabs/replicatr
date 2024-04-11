package badger

import (
	"strconv"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/arb"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/kinder"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/pubkey"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"mleku.dev/git/ec/schnorr"
)

// GetTagKeyPrefix returns tag index prefixes based on the initial field of a
// tag.
//
// There is 3 types of index tag keys:
//
// - TagAddr:   [ 8 ][ 2b Kind ][ 8b Pubkey ][ address/URL ][ 8b Serial ]
// - Tag32:     [ 7 ][ 8b Pubkey ][ 8b Serial ]
// - Tag:       [ 6 ][ address/URL ][ 8b Serial ]
//
// This function produces the initial bytes without the index.
func GetTagKeyPrefix(tagValue string) (key []byte, err error) {
	// log.T.Ln("tagValue", tagValue)
	if k, pkb, d := eventstore.GetAddrTagElements(tagValue); len(pkb) == 32 {
		// // store value in the new special "a" tag index
		// offset = prefix.Len + kinder.Len + pubkey.Len + len(d)
		// log.T.F("k %d pkb %x d '%s'", k, pkb, d)
		var pk *pubkey.T
		pk, err = pubkey.NewFromBytes(pkb)
		els := []keys.Element{kinder.New(k), pk}
		if len(d) > 0 {
			els = append(els, arb.NewFromString(d))
		}
		key = index.TagAddr.Key(els...)
		// key = make([]byte, 1+2+8+len(d)+4+SerialLen)
		// key[0] = prefixes.TagAddr
		// binary.BigEndian.PutUint16(key[1:], k)
		// copy(key[1+2:], pkb[0:8])
		// copy(key[1+2+8:], d)
	} else if pkb, _ := hex.Dec(tagValue); len(pkb) == 32 {
		// store value as bytes
		// offset = 1 + 8
		// key = make([]byte, 1+8+4+SerialLen)
		// key[0] = prefixes.Tag32
		// copy(key[1:], vb[0:8])
		var pkk *pubkey.T
		if pkk, err = pubkey.NewFromBytes(pkb); chk.E(err) {
			return
		}
		key = index.Tag32.Key(pkk)
	} else {
		// store whatever as utf-8
		// offset = 1 + len(tagValue)
		// key = make([]byte, 1+len(tagValue)+4+SerialLen)
		// key[0] = prefixes.Tag
		// copy(key[1:], tagValue)
		if len(tagValue) > 0 {
			var a *arb.T
			a = arb.NewFromString(tagValue)
			key = index.Tag.Key(a)
		}
		key = index.Tag.Key()
	}
	return
}

func GetTagKeyElements(tagValue string) (prf index.P, elems []keys.Element) {
	if len(tagValue) == 2*schnorr.PubKeyBytesLen {
		// this could be a pubkey
		pkb, err := hex.Dec(tagValue)
		if err == nil {
			// it's a pubkey
			var pkk keys.Element
			if pkk, err = pubkey.NewFromBytes(pkb); chk.E(err) {
				return
			}
			return index.Tag32, []keys.Element{pkk}
		}
	}
	// check for a tag
	if strings.Count(tagValue, ":") == 2 {
		split := strings.Split(tagValue, ":")
		if len(split) == 3 {
			var k uint16
			var pkb []byte
			var d string
			_ = d
			if pkb, _ = hex.Dec(split[1]); len(pkb) == 32 {
				if key, err := strconv.ParseUint(split[0], 10, 16); err == nil {
					k = uint16(key)
					d = split[2]
					var pk *pubkey.T
					pk, err = pubkey.NewFromBytes(pkb)
					return index.TagAddr, []keys.Element{kinder.New(k), pk}
				}
			}
		}
	}
	return
}
