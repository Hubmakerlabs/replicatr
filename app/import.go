package app

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
)

// Import a collection of JSON events from stdin or from one or more files, line
// structured JSON.
func (rl *Relay) Import(db eventstore.Store, files []string, wg *sync.WaitGroup, start int) {
	wg.Add(1)
	defer wg.Done()
	log.D.Ln("running import subcommand on these files:", files)
	var err error
	var fh *os.File
	buf := make([]byte, rl.MaxMessageSize)
	for i := range files {
		log.I.Ln("importing from file", files[i])
		if fh, err = os.OpenFile(files[i], os.O_RDONLY, 0755); chk.D(err) {
			continue
		}
		if start != 0 {
			_, err = fh.Seek(int64(start), 0)
			chk.E(err)
		}
		scanner := bufio.NewScanner(fh)
		scanner.Buffer(buf, 500000000)
		var counter int
		for scanner.Scan() {
			select {
			case <-rl.Ctx.Done():
				return
			default:
			}
			b := scanner.Bytes()
			counter++
			ev := &event.T{}
			if err = json.Unmarshal(b, ev); chk.E(err) {
				continue
			}
			log.I.Ln(counter, ev.ToObject().String())
			if ev.Kind == kind.Deletion {
				// this always returns "blocked: " whenever it returns an error
			} else {
				// this will also always return a prefixed reason
				err = db.SaveEvent(context.Bg(), ev)
			}
		}
		chk.D(fh.Close())
	}
}
