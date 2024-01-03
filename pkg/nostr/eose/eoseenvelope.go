package eose

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
	log2 "mleku.online/git/log"
)

var log = log2.GetLogger()
var fails = log.D.Chk

// Envelope is a message that indicates that all cached events have been
// delivered and thereafter events will be new and delivered in pubsub subscribe
// fashion while the socket remains open.
type Envelope struct {
	subscriptionid.T
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.List[T]
func (E *Envelope) Label() (l labels.T) { return labels.LEOSE }

func (E *Envelope) ToArray() (a array.T) {
	a = array.T{labels.EOSE, E.T}
	return
}

func (E *Envelope) String() (s string) {
	return E.ToArray().String()
}

func (E *Envelope) Bytes() (s []byte) {
	return E.ToArray().Bytes()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *Envelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *Envelope) Unmarshal(buf *text.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label (note we aren't checking that only
	// whitespace intervenes because laziness, usually this is the very next
	// character).
	if e = buf.ScanUntil(','); e != nil {
		return
	}
	// Next character we find will be open quotes for the subscription ID.
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var sid []byte
	// read the string
	if sid, e = buf.ReadUntil('"'); fails(e) {
		return fmt.Errorf("unterminated quotes in JSON, probably truncated read")
	}
	E.T = subscriptionid.T(sid[:])
	return
}
