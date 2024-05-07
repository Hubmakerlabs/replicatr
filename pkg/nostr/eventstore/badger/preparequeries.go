package badger

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/kinder"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/pubkey"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

type query struct {
	index        int
	queryFilter  *filter.T
	searchPrefix []byte
	start        []byte
	results      chan Results
	skipTS       bool
}

func (q query) String() string {
	return fmt.Sprintf("%d %s prf %0x start %0x skipTS %v", q.index,
		// text.Trunc(
		q.queryFilter.ToObject().String(),
		// ),
		q.searchPrefix,
		q.start,
		q.skipTS,
	)
}

type Results struct {
	Ev  *event.T
	TS  timestamp.T
	Ser *serial.T
}

// PrepareQueries analyses a filter and generates a set of query specs that produce
// key prefixes to search for in the badger key indexes.
func PrepareQueries(f *filter.T) (
	qs []query,
	ext *filter.T,
	since uint64,
	err error,
) {
	switch {
	// first if there is IDs, just search for them, this overrides all other filters
	case len(f.IDs) > 0:
		qs = make([]query, len(f.IDs))
		for i, idHex := range f.IDs {
			ih := id.New(idHex)
			if ih == nil {
				log.E.F("failed to decode event ID: %s", idHex)
				// just ignore it, clients will be clients
				continue
			}
			prf := index.Id.Key(ih)
			// log.T.F("id prefix to search on %0x from key %0x", prf, ih.Val)
			qs[i] = query{
				index:        i,
				queryFilter:  f,
				searchPrefix: prf,
				skipTS:       true, // why are we not checking timestamps? (ID has no timestamp)
			}
		}
		// log.T.S("ids", qs)
		// second we make a set of queries based on author pubkeys, optionally with kinds
	case len(f.Authors) > 0:
		// if there is no kinds, we just make the queries based on the author pub keys
		if len(f.Kinds) == 0 {
			qs = make([]query, len(f.Authors))
			for i, pubkeyHex := range f.Authors {
				var pk *pubkey.T
				if pk, err = pubkey.New(pubkeyHex); chk.E(err) {
					// bogus filter, continue anyway
					continue
				}
				sp := index.Pubkey.Key(pk)
				// log.I.F("search only for authors %0x from pub key %0x", sp, pk.Val)
				qs[i] = query{
					index:        i,
					queryFilter:  f,
					searchPrefix: sp,
				}
			}
			// log.I.S("authors", qs)
		} else {
			// if there is kinds as well, we are searching via the kind/pubkey prefixes
			qs = make([]query, len(f.Authors)*len(f.Kinds))
			i := 0
		authors:
			for _, pubkeyHex := range f.Authors {
				for _, kind := range f.Kinds {
					var pk *pubkey.T
					if pk, err = pubkey.New(pubkeyHex); chk.E(err) {
						// skip this dodgy thing
						continue authors
					}
					ki := kinder.New(kind)
					sp := index.PubkeyKind.Key(pk, ki)
					// log.T.F("search for authors from pub key %0x and kind %0x", pk.Val, ki.Val)
					qs[i] = query{index: i, queryFilter: f, searchPrefix: sp}
					i++
				}
			}
			// log.T.S("authors/kinds", qs)
		}
		if f.Tags != nil || len(f.Tags) > 0 {
			ext = &filter.T{Tags: f.Tags}
			// log.T.S("extra filter", ext)
		}
	case len(f.Tags) > 0:
		// determine the size of the queries array by inspecting all tags sizes
		size := 0
		for _, values := range f.Tags {
			size += len(values)
		}
		if size == 0 {
			return nil, nil, 0, fmt.Errorf("empty tag filters")
		}
		// we need a query for each tag search
		qs = make([]query, size)
		// and any kinds mentioned as well in extra filter
		ext = &filter.T{Kinds: f.Kinds}
		i := 0
		for _, values := range f.Tags {
			for _, value := range values {
				// get key prefix (with full length) and offset where to write the last parts
				var prf []byte
				if prf, err = GetTagKeyPrefix(value); chk.E(err) {
					continue
				}
				// remove the last part to get just the prefix we want here
				// log.T.F("search for tags from %0x", prf)
				qs[i] = query{index: i, queryFilter: f, searchPrefix: prf}
				i++
			}
		}
		// log.T.S("tags", qs)
	case len(f.Kinds) > 0:
		// if there is no ids, pubs or tags, we are just searching for kinds
		qs = make([]query, len(f.Kinds))
		for i, kind := range f.Kinds {
			kk := kinder.New(kind)
			ki := index.Kind.Key(kk)
			qs[i] = query{
				index:        i,
				queryFilter:  f,
				searchPrefix: ki,
			}
		}
		// log.T.S("kinds", qs)
	default:
		if len(qs) > 0 {
			qs[0] = query{index: 0, queryFilter: f, searchPrefix: index.CreatedAt.Key()}
			ext = nil
			// log.T.S("other", qs)
		}
	}
	var until uint64 = math.MaxUint64
	if f.Until != nil {
		if fu := uint64(*f.Until); fu < until {
			until = fu + 1
		}
	}
	for i, q := range qs {
		qs[i].start = binary.BigEndian.AppendUint64(q.searchPrefix, until)
		qs[i].results = make(chan Results, 128)
	}
	// this is where we'll end the iteration
	if f.Since != nil {
		if fs := uint64(*f.Since); fs > since {
			since = fs
		}
	}
	return
}
