package app

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/minio/sha256-simd"
)

// Import a collection of JSON events from stdin or from one or more files, line
// structured JSON.
func (rl *Relay) Import(db *badger.Backend, files []string) {
	log.D.Ln("running export subcommand on these files:", files)
	var err error
	var fh *os.File
	buf := make([]byte, rl.MaxMessageSize)
	for i := range files {
		if fh, err = os.OpenFile(files[i], os.O_RDONLY, 0755); chk.D(err) {
			continue
		}
		scanner := bufio.NewScanner(fh)
		scanner.Buffer(buf, int(rl.MaxMessageSize))
		for scanner.Scan() {
			b := scanner.Bytes()
			ev := &event.T{}
			if err = json.Unmarshal(b, ev); chk.D(err) {
				log.D.S(string(b))
				continue
			}
			evb := ev.ToCanonical().Bytes()
			hash := sha256.Sum256(evb)
			id := hex.Enc(hash[:])
			if id != ev.ID.String() {
				log.D.F("id mismatch got %s, expected %s", id, ev.ID.String())
				continue
			}
			log.D.Ln("ID was valid")
			// check signature
			var ok bool
			if ok, err = ev.CheckSignature(); chk.E(err) {
				log.E.F("error: failed to verify signature: %v", err)
				continue
			} else if !ok {
				log.E.Ln("invalid: signature is invalid")
				return
			}
			log.D.Ln("signature was valid")
			if ev.Kind == kind.Deletion {
				// this always returns "blocked: " whenever it returns an error
				err = rl.handleDeleteRequest(context.Bg(), ev)
			} else {
				log.D.Ln("adding event")
				// this will also always return a prefixed reason
				err = rl.AddEvent(context.Bg(), ev)
			}
			chk.E(err)
		}
		chk.D(fh.Close())
	}
}
