package app

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nostrbinary"
	bdb "github.com/dgraph-io/badger/v4"
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
	chk.E(db.View(func(txn *bdb.Txn) (err error) {
		prf := index.Event.Key()
		it := txn.NewIterator(bdb.IteratorOptions{Prefix: prf})
		for it.Rewind(); it.ValidForPrefix(prf); it.Next() {
			if b, err = it.Item().ValueCopy(b); err != nil {
				continue
			}
			if len(b) == 0 {
				continue
			}
			// log.D.S(b)
			buf := bytes.NewBuffer(b)
			dec := gob.NewDecoder(buf)
			ev := &nostrbinary.Event{}
			if err = dec.Decode(ev); chk.E(err) {
				continue
			}
			buf.Reset()
			fmt.Fprintln(fh, ev.ToEventT().ToObject().String())
		}
		it.Close()
		return nil
	}))
}
