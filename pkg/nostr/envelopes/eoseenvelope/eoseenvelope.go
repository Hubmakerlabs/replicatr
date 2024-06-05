package eoseenvelope

import (
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"mleku.net/slog"
)

// const RELAY = "wss://nos.lol"

var log, chk = slog.New(os.Stderr)

// T is a message that indicates that all cached events have been delivered and
// thereafter events will be new and delivered in publish/subscribe fashion
// while the socket remains open.
type T struct {
	Sub subscriptionid.T
}

var _ enveloper.I = (*T)(nil)

func (E *T) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

func (E *T) Label() string { return labels.EOSE }

func (E *T) ToArray() array.T { return array.T{labels.EOSE, E.Sub} }

func (E *T) String() (s string) { return E.ToArray().String() }

func (E *T) Bytes() (s []byte) { return E.ToArray().Bytes() }

func (E *T) MarshalJSON() ([]byte, error) { return E.Bytes(), nil }

// Unmarshal the envelope.
func (E *T) Unmarshal(buf *text.Buffer) (err error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label (note we aren't checking that only
	// whitespace intervenes because laziness, usually this is the very next
	// character).
	if err = buf.ScanUntil(','); err != nil {
		return
	}
	// Next character we find will be open quotes for the subscription ID.
	if err = buf.ScanThrough('"'); err != nil {
		return
	}
	var sid []byte
	// read the string
	if sid, err = buf.ReadUntil('"'); chk.D(err) {
		return fmt.Errorf("unterminated quotes in JSON, probably truncated read: %s",
			err)
	}
	E.Sub = subscriptionid.T(sid[:])
	return
}
