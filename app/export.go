package app

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	bdb "github.com/dgraph-io/badger/v4"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
)

// Export prints the JSON of all events or writes them to a file.
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
	var evs event.Array
	chk.E(db.View(func(txn *bdb.Txn) (err error) {
		it := txn.NewIterator(bdb.IteratorOptions{})
		for it.Rewind(); it.Valid(); it.Next() {
			if b, err = it.Item().ValueCopy(b); err != nil {
				continue
			}
			if len(b) == 0 {
				continue
			}
			// log.D.S(b)
			buf := bytes.NewBuffer(b)
			dec := gob.NewDecoder(buf)
			ev := &event.T{}
			if err = dec.Decode(ev); err != nil {
				continue
			}
			buf.Reset()
			evs = append(evs, ev)
		}
		it.Close()
		return nil
	}))
	sort.Sort(evs)
	for i := range evs {
		fmt.Fprintln(fh, evs[i].ToObject().String())
	}
}
