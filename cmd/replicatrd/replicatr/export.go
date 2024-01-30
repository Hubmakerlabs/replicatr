package replicatr

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	bdb "github.com/dgraph-io/badger/v4"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
)

// Export prints the JSON of all events or writes them to a file.
func (rl *Relay) Export(db *badger.Backend, filename string) {
	rl.D.Ln("running export subcommand")
	b := make([]byte, MaxMessageSize)
	ev := &event.T{}
	gob.Register(ev)
	var fh *os.File
	var err error
	if filename != "" {
		fh, err = os.Open(filename)
		if err != nil {
			rl.F.Ln(err)
			os.Exit(1)
		}
	} else {
		fh = os.Stdout
	}
	rl.E.Chk(db.View(func(txn *bdb.Txn) (err error) {
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
			if err = dec.Decode(ev); err != nil {
				continue
			}
			buf.Reset()
			fmt.Fprintln(fh, ev.ToObject().String())
		}
		it.Close()
		return nil
	}))
}
