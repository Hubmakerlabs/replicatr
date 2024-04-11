// Package nostrbinary provides a simple interface for using Gob encoding on
// nostr events.
package nostrbinary

import (
	"bytes"
	"encoding/gob"
	"os"

	"mleku.dev/git/nostr/event"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

func Unmarshal(data []byte) (evt *event.T, err error) {

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	evt = &event.T{}
	if err = dec.Decode(evt); chk.D(err) {
		return
	}
	return
}

func Marshal(evt *event.T) (b []byte, err error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err = enc.Encode(evt); chk.D(err) {
		return
	}
	b = buf.Bytes()
	return
}
