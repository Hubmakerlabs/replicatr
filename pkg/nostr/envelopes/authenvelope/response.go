package authenvelope

import (
	"encoding/json"
	"fmt"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/enveloper"
	l "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

var log = log2.GetStd()

type Response struct {
	*event.T
}

var _ enveloper.I = &Response{}

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

func (a *Response) Label() string { return l.AUTH }

func (a *Response) ToArray() array.T { return array.T{l.AUTH, a.T.ToObject()} }

func (a *Response) String() string { return a.ToArray().String() }

func (a *Response) Bytes() []byte { return a.ToArray().Bytes() }

func (a *Response) MarshalJSON() ([]byte, error) { return a.Bytes(), nil }

func (a *Response) Unmarshal(buf *text.Buffer) (e error) {
	if a == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label (note we aren't checking that only
	// whitespace intervenes because laziness, usually this is the very next
	// character).
	if e = buf.ScanUntil(','); e != nil {
		return fmt.Errorf("event field not found in auth envelope")
	}
	// find the opening brace of the event object, usually this is the very next
	// character, we aren't checking for valid whitespace because laziness.
	if e = buf.ScanUntil('{'); e != nil {
		return fmt.Errorf("event not found in auth envelope")
	}
	// now we should have an event object next. It has no embedded object so it
	// should end with a close brace. This slice will be wrapped in braces and
	// contain paired brackets, braces and quotes.
	var eventObj []byte
	if eventObj, e = buf.ReadEnclosed(); log.Fail(e) {
		return fmt.Errorf("event not found in auth envelope")
	}
	// allocate an event to unmarshal into
	a.T = &event.T{}
	if e = json.Unmarshal(eventObj, a.T); log.Fail(e) {
		log.D.S(string(eventObj))
		return
	}
	// technically we maybe should read ahead further to make sure the JSON
	// closes correctly. Not going to abort because of this.
	if e = buf.ScanUntil(']'); e != nil {
		return fmt.Errorf("malformed JSON, no closing bracket on array")
	}
	// whatever remains doesn't matter as the envelope has fully unmarshaled.
	return
}
