package replicatr

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/minio/sha256-simd"
)

// Import a collection of JSON events from stdin or from one or more files, line
// structured JSON.
func (rl *Relay) Import(db *badger.Backend, files []string) {
	rl.D.Ln("running export subcommand on these files:", files)
	var err error
	var fh *os.File
	buf := make([]byte, rl.MaxMessageSize)
	for i := range files {
		if fh, err = os.OpenFile(files[i], os.O_RDONLY, 0755); rl.Fail(err) {
			continue
		}
		scanner := bufio.NewScanner(fh)
		scanner.Buffer(buf, int(rl.MaxMessageSize))
		for scanner.Scan() {
			b := scanner.Bytes()
			ev := &event.T{}
			if err = json.Unmarshal(b, ev); rl.Fail(err) {
				rl.D.S(string(b))
				continue
			}
			evb := ev.ToCanonical().Bytes()
			hash := sha256.Sum256(evb)
			id := hex.Enc(hash[:])
			if id != ev.ID.String() {
				rl.T.F("id mismatch got %s, expected %s", id, ev.ID.String())
				continue
			}
			rl.T.Ln("ID was valid")
			// check signature
			var ok bool
			if ok, err = ev.CheckSignature(); rl.E.Chk(err) {
				rl.E.F("error: failed to verify signature: %v", err)
				continue
			} else if !ok {
				rl.E.Ln("invalid: signature is invalid")
				return
			}
			rl.T.Ln("signature was valid")
			if ev.Kind == kind.Deletion {
				// this always returns "blocked: " whenever it returns an error
				err = rl.handleDeleteRequest(context.Bg(), ev)
			} else {
				rl.D.Ln("adding event")
				// this will also always return a prefixed reason
				err = rl.AddEvent(context.Bg(), ev)
			}
			rl.E.Chk(err)
		}
		rl.Fail(fh.Close())
	}
}
