package eventenvelope

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

var _ enveloper.I = (*T)(nil)

// T is the wrapper expected by a relay around an event.
type T struct {
	// The SubscriptionID field is optional, and may at most contain 64 characters,
	// sufficient for encoding a 256 bit hash as hex.
	SubscriptionID subscriptionid.T
	// The Event is here a pointer because it should not be copied unnecessarily.
	Event *event.T
}

// NewEventEnvelope builds an T from a provided T string and pointer to an T,
// and returns either the T or an error if the Subscription ID is invalid or the
// T is nil.
func NewEventEnvelope(si string, ev *event.T) (ee *T, err error) {
	var sid subscriptionid.T
	if sid, err = subscriptionid.New(si); chk.D(err) {
		return
	}
	if ev == nil {
		err = log.E.Err("cannot make event envelope with nil event")
		return
	}
	return &T{SubscriptionID: sid, Event: ev}, nil
}

func FromRawJSON(si string, j []byte) (b []byte) {
	// pre-allocate all the memory the buffer will need.
	buf := bytes.NewBuffer(make([]byte, len(si)+len(j)+16))
	buf.WriteString(`["EVENT","`)
	buf.WriteString(si)
	buf.WriteString(`",`)
	buf.Write(j)
	buf.WriteString(`]`)
	return buf.Bytes()
}

func (env *T) ToArray() (a array.T) {
	a = make(array.T, 0, 3)
	a = append(a, labels.EVENT)
	if env.SubscriptionID.IsValid() {
		a = append(a, env.SubscriptionID)
	}
	a = append(a, env.Event.ToObject())
	return
}

func (env *T) Label() string                { return labels.EVENT }
func (env *T) String() (s string)           { return env.ToArray().String() }
func (env *T) Bytes() (s []byte)            { return env.ToArray().Bytes() }
func (env *T) MarshalJSON() ([]byte, error) { return env.Bytes(), nil }

// Unmarshal the envelope.
func (env *T) Unmarshal(buf *text.Buffer) (err error) {
	if env == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label (note we aren't checking that only
	// whitespace intervenes because laziness, usually this is the very next
	// character).
	if err = buf.ScanUntil(','); log.D.Chk(err) {
		return
	}
	// Next character we find will be open quotes for the subscription ID, or
	// the open brace of the embedded event.
	var matched byte
	if matched, err = buf.ScanForOneOf(false, '"', '{'); log.D.Chk(err) {
		return
	}
	if matched == '"' {
		// Advance the cursor to consume the quote character.
		buf.Pos++
		var sid []byte
		// Read the string.
		if sid, err = buf.ReadUntil('"'); log.D.Chk(err) {
			return fmt.Errorf("unterminated quotes in JSON, probably truncated read")
		}
		env.SubscriptionID = subscriptionid.T(sid[:])
		// Next, find the comma after the subscription ID (note we aren't checking
		// that only whitespace intervenes because laziness, usually this is the
		// very next character).
		if err = buf.ScanUntil(','); log.D.Chk(err) {
			return fmt.Errorf("event not found in event envelope")
		}
	}
	// find the opening brace of the event object, usually this is the very next
	// character, we aren't checking for valid whitespace because laziness.
	if err = buf.ScanUntil('{'); log.D.Chk(err) {
		return fmt.Errorf("event not found in event envelope")
	}
	// now we should have an event object next. It has no embedded object so it
	// should end with a close brace. This slice will be wrapped in braces and
	// contain paired brackets, braces and quotes.
	var eventObj []byte
	if eventObj, err = buf.ReadEnclosed(); chk.D(err) {
		fmt.Println(string(buf.Buf))
		return
	}
	// allocate an event to unmarshal into
	env.Event = &event.T{}
	if err = json.Unmarshal(eventObj, env.Event); chk.D(err) {
		log.D.Ln(string(eventObj))
		return
	}
	// technically we maybe should read ahead further to make sure the JSON closes
	// correctly. Not going to abort because of this. whatever remains doesn't
	// matter as the envelope has fully unmarshalled.
	return
}
