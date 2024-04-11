package app

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"os"
	"sort"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
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
	_ = fh
	var exps ExportEntries
	// first gather the last accessed timestamps
	chk.E(db.View(func(txn *bdb.Txn) (err error) {
		it := txn.NewIterator(bdb.IteratorOptions{})
		for it.Rewind(); it.Valid(); it.Next() {
			var k []byte
			if k = it.Item().KeyCopy(nil); err != nil {
				continue
			}
			switch k[0] {
			// case index.Event.Byte():
			// 	// get the event
			// 	buf := bytes.NewBuffer(b)
			// 	dec := gob.NewDecoder(buf)
			// 	ev := &event.T{}
			// 	if err = dec.Decode(ev); err != nil {
			// 		continue
			// 	}
			// 	buf.Reset()
			// 	// add to the list of entries to be exported
			// 	ex := exps.Find(k)
			// 	if ex != nil {
			// 		ex.ev = ev
			// 	} else {
			// 		exps = append(exps, ExportEntry{idx: k, ev: ev})
			// 	}
			case index.Counter.Byte():
				log.I.F("k %x ser %x", k[0], k[9:])
				if b, err = it.Item().ValueCopy(b); err != nil {
					continue
				}
				// get the last accessed timestamp
				la := timestamp.T(binary.BigEndian.Uint64(b[:createdat.Len]))
				exps = append(exps, ExportEntry{idx: k[1:], lastAccessed: la})
			default:
				// not interesting
				// continue
			}
		}
		it.Close()
		return nil
	}))
	// // sort list of entries by last accessed timestamp
	// sort.Sort(exps)
	// log.I.S(exps)
	// // now output the entries in this order
	// last := len(exps)
	// var i int
	// for ; i < last; i++ {
	// 	chk.E(db.View(func(txn *bdb.Txn) (err error) {
	// 		it := txn.NewIterator(bdb.IteratorOptions{})
	// 		defer it.Close()
	// 		it.Seek(append([]byte{index.Event.Byte()}, exps[i].idx...))
	// 		if it.Valid() {
	// 			log.I.F("%x %x", exps[i].idx, it.Item().KeyCopy(nil))
	// 			if b, err = it.Item().ValueCopy(nil); chk.E(err) {
	// 				return
	// 			}
	// 			log.I.S(b)
	// 			buf := new(bytes.Buffer)
	// 			dec := gob.NewDecoder(buf)
	// 			ev := &event.T{}
	// 			if err = dec.Decode(ev); chk.E(err) {
	// 				return nil
	// 			}
	// 			log.I.Ln(exps[i].lastAccessed.Time())
	// 			fmt.Fprintln(fh, ev.ToObject().String())
	// 		}
	// 		return nil
	// 	}))
	// }
}
