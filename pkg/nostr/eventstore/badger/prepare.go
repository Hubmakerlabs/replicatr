package badger

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/kinder"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/pubkey"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
)

func prepareQueries(f *filter.T) (
	qs []query,
	ext *filter.T,
	since uint64,
	err error,
) {
	// var index byte
	if len(f.IDs) > 0 {
		// index = indexIdPrefix
		qs = make([]query, len(f.IDs))
		for i, idHex := range f.IDs {
			// prefix := make([]byte, 1+8)
			// prefix[0] = index
			// if len(idHex) != 64 {
			// 	return nil, nil, 0, fmt.Errorf("invalid id '%s'", idHex)
			// }
			// idPrefix8, _ := hex.DecodeString(idHex[0 : 8*2])
			// copy(prefix[1:], idPrefix8)
			// qs[i] = query{i: i, prefix: prefix, skipTS: true}
			prf := index.Id.Key(id.New(idHex))
			qs[i] = query{i: i, f: f, prefix: prf, skipTS: true}
		}
	} else if len(f.Authors) > 0 {
		if len(f.Kinds) == 0 {
			// index = indexPubkeyPrefix
			qs = make([]query, len(f.Authors))
			for i, pubkeyHex := range f.Authors {
				// if len(pubkeyHex) != 64 {
				// 	return nil, nil, 0, fmt.Errorf("invalid pubkey '%s'", pubkeyHex)
				// }
				// pubkeyPrefix8, _ := hex.DecodeString(pubkeyHex[0 : 8*2])
				// prefix := make([]byte, 1+8)
				// prefix[0] = index
				// copy(prefix[1:], pubkeyPrefix8)
				// qs[i] = query{i: i, prefix: prefix}
				var pk *pubkey.T
				if pk, err = pubkey.New(pubkeyHex); chk.E(err) {
					return
				}
				qs[i] = query{i: i, f: f, prefix: index.Pubkey.Key(pk)}
			}
		} else {
			// index = indexPubkeyKindPrefix
			qs = make([]query, len(f.Authors)*len(f.Kinds))
			i := 0
			for _, pubkeyHex := range f.Authors {
				for _, kind := range f.Kinds {
					// if len(pubkeyHex) != 64 {
					// 	return nil, nil, 0, fmt.Errorf("invalid pubkey '%s'", pubkeyHex)
					// }
					// pubkeyPrefix8, _ := hex.DecodeString(pubkeyHex[0 : 8*2])
					// prefix := make([]byte, 1+8+2)
					// prefix[0] = index
					// copy(prefix[1:], pubkeyPrefix8)
					// binary.BigEndian.PutUint16(prefix[1+8:], uint16(kind))
					// qs[i] = query{i: i, prefix: prefix}
					var pk *pubkey.T
					if pk, err = pubkey.New(pubkeyHex); chk.E(err) {
						return
					}
					qs[i] = query{i: i, f: f,
						prefix: index.PubkeyKind.Key(
							pk, kinder.New(kind),
						)}
					i++
				}
			}
		}
		ext = &filter.T{Tags: f.Tags}
	} else if len(f.Tags) > 0 {
		// determine the size of the queries array by inspecting all tags sizes
		size := 0
		for _, values := range f.Tags {
			size += len(values)
		}
		if size == 0 {
			return nil, nil, 0, fmt.Errorf("empty tag filters")
		}
		qs = make([]query, size)
		ext = &filter.T{Kinds: f.Kinds}
		i := 0
		for _, values := range f.Tags {
			for _, value := range values {
				// get key prefix (with full length) and offset where to write the last parts
				k, _ := getTagIndexPrefix(value)
				// remove the last part to get just the prefix we want here
				// prefix := k[0:offset]

				qs[i] = query{i: i, f: f, prefix: k}
				i++
			}
		}
	} else if len(f.Kinds) > 0 {
		// index = indexKindPrefix
		qs = make([]query, len(f.Kinds))
		for i, kind := range f.Kinds {
			// prefix := make([]byte, 1+2)
			// prefix[0] = index
			// binary.BigEndian.PutUint16(prefix[1:], uint16(kind))
			// qs[i] = query{i: i, prefix: prefix}
			qs[i] = query{i: i, f: f,
				prefix: index.Kind.Key(kinder.New(kind))}

		}
	} else {
		// index = indexCreatedAtPrefix
		// qs = make([]query, 1)
		// prefix := make([]byte, 1)
		// prefix[0] = index
		// qs[0] = query{i: 0, prefix: prefix}
		qs[0] = query{i: 0, f: f, prefix: index.CreatedAt.Key()}
		ext = nil
	}

	var until uint64 = math.MaxUint64
	if f.Until != nil {
		if fu := uint64(*f.Until); fu < until {
			until = fu + 1
		}
	}
	for i, q := range qs {
		qs[i].start = binary.BigEndian.AppendUint64(q.prefix, until)
		qs[i].results = make(chan Results, 12)
		// qs[i].results = make(chan *event.T, 12)
	}

	// this is where we'll end the iteration
	if f.Since != nil {
		if fs := uint64(*f.Since); fs > since {
			since = fs
		}
	}

	return qs, ext, since, nil
}
