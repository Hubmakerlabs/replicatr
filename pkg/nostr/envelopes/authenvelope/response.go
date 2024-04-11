package authenvelope

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Response struct {
	Event *event.T
}

var _ enveloper.I = &Response{}

func (a *Response) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

// NewResponse creates an Response response from an Challenge.
//
// The caller must sign the embedded event before sending it back to
// authenticate.
func NewResponse(ac *Challenge, rl string) (ae *Response) {
	ae = &Response{
		&event.T{
			Kind: kind.ClientAuthentication,
			Tags: tags.T{{"relay", rl}, {"challenge", ac.Challenge}},
		},
	}
	return
}

func (a *Response) Label() string { return labels.AUTH }

func (a *Response) ToArray() array.T { return array.T{labels.AUTH, a.Event.ToObject()} }

func (a *Response) String() string { return a.ToArray().String() }

func (a *Response) Bytes() []byte { return a.ToArray().Bytes() }

func (a *Response) MarshalJSON() ([]byte, error) { return a.Bytes(), nil }

func (a *Response) Unmarshal(buf *text.Buffer) (err error) {
	// log.T.F("AUTH '%s'", buf.Tail())
	if a == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label (note we aren't checking that only
	// whitespace intervenes because laziness, usually this is the very next
	// character).
	if err = buf.ScanUntil(','); err != nil {
		return fmt.Errorf("event field not found in auth envelope")
	}
	// find the opening brace of the event object, usually this is the very next
	// character, we aren't checking for valid whitespace because laziness.
	if err = buf.ScanUntil('{'); err != nil {
		return fmt.Errorf("event not found in auth envelope")
	}
	// now we should have an event object next. It has no embedded object so it
	// should end with a close brace. This slice will be wrapped in braces and
	// contain paired brackets, braces and quotes.
	var eventObj []byte
	if eventObj, err = buf.ReadEnclosed(); chk.D(err) {
		return fmt.Errorf("event not found in auth envelope: %s", err)
	}
	// allocate an event to unmarshal into
	a.Event = &event.T{}
	if err = json.Unmarshal(eventObj, a.Event); chk.D(err) {
		log.D.S(string(eventObj))
		return
	}
	// technically we maybe should read ahead further to make sure the JSON
	// closes correctly. Not going to abort because of this.
	if err = buf.ScanUntil(']'); err != nil {
		return fmt.Errorf("malformed JSON, no closing bracket on array: %s", err)
	}
	// whatever remains doesn't matter as the envelope has fully unmarshaled.
	return
}
