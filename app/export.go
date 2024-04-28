package app

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	bdb "github.com/dgraph-io/badger/v4"
)

type ExportEntry struct {
	idx          []byte
	lastAccessed timestamp.T
}

type ExportEntries []ExportEntry

// sort export entries the same way as the GC sorts them for pruning
var _ sort.Interface = ExportEntries{}

func (e ExportEntries) Len() int           { return len(e) }
func (e ExportEntries) Less(i, j int) bool { return e[i].lastAccessed < e[j].lastAccessed }
func (e ExportEntries) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

func (e ExportEntries) Find(idx []byte) (ee *ExportEntry) {
	for i := range e {
		if bytes.Compare(e[i].idx, idx) == 0 {
			return &e[i]
		}
	}
	return
}

// Export prints the JSON of all events or writes them to a file.
//
// This is a simple, direct query, no sorting or filtering is done, it simply
// prints all events in the event store.
func (rl *Relay) Export(db *badger.Backend, filename string) {
	log.D.Ln("running export subcommand")
	b := make([]byte, MaxMessageSize)
	gob.Register(&event.T{})
	var fh *os.File
	var err error
	if filename != "" {
		fh, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
		if chk.F(err) {
			os.Exit(1)
		}
	} else {
		fh = os.Stdout
	}
	chk.E(db.View(func(txn *bdb.Txn) (err error) {
		prf := index.Event.Key()
		it := txn.NewIterator(bdb.IteratorOptions{Prefix: prf})
		for it.Rewind(); it.ValidForPrefix(prf); it.Next() {
			if _, err = it.Item().ValueCopy(b); chk.E(err) {
				continue
			}
			// get the event
			buf := bytes.NewBuffer(b)
			var ev *event.T
			if ev, err = nostrbinary.Unmarshal(b); chk.E(err) {
				continue
			}
			buf.Reset()
			// print to output
			_, _ = fmt.Fprintln(fh, ev.ToObject().String())
		}
		it.Close()
		return nil
	}))
}
