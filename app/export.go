package app

import (
	"bytes"
	"encoding/gob"
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
	prf := []byte{index.Event.Byte()}
	// first gather the last accessed timestamps
	chk.E(db.View(func(txn *bdb.Txn) (err error) {
		it := txn.NewIterator(bdb.IteratorOptions{Prefix: prf})
		var ev *event.T
		for it.Rewind(); it.ValidForPrefix(prf); it.Next() {
			// get the event
			if b, err = it.Item().ValueCopy(b); chk.E(err) {
				continue
			}
			buf := bytes.NewBuffer(b)
			if ev, err = nostrbinary.Unmarshal(b); chk.E(err) {
				continue
			}
			buf.Reset()
			if _, err = fh.Write(ev.ToObject().Bytes()); chk.E(err) {
				continue
			}
			if _, err = fh.Write([]byte("\n")); chk.E(err) {
				continue
			}
		}
		it.Close()
		return nil
	}))
}
