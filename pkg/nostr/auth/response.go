package auth

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
	log2 "mleku.online/git/log"
)

var (
	log                    = log2.GetLogger()
	fails                  = log.D.Chk
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

type ResponseEnvelope struct {
	*event.T
}

// New creates an ResponseEnvelope response from an ChallengeEnvelope.
//
// The caller must sign the embedded event before sending it back to
// authenticate.
func New(ac *ChallengeEnvelope, rl string) (ae *ResponseEnvelope) {
	ae = &ResponseEnvelope{
		&event.T{
			Kind: kind.ClientAuthentication,
			Tags: tags.T{
				{"relay", rl},
				{"challenge", ac.Challenge},
			},
		},
	}
	return
}

func (a *ResponseEnvelope) Label() labels.T { return labels.LAuth }

func (a *ResponseEnvelope) Unmarshal(buf *text.Buffer) (e error) {
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
	if eventObj, e = buf.ReadEnclosed(); fails(e) {
		return fmt.Errorf("event not found in auth envelope")
	}
	// allocate an event to unmarshal into
	a.T = &event.T{}
	if e = json.Unmarshal(eventObj, a.T); fails(e) {
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

func (a *ResponseEnvelope) ToArray() array.T {
	return array.T{labels.List[labels.LAuth], a.T.ToObject()}
}

func (a *ResponseEnvelope) String() (s string) {
	return a.ToArray().String()
}

func (a *ResponseEnvelope) Bytes() (s []byte) {
	return a.ToArray().Bytes()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (a *ResponseEnvelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("auth envelope marshal")
	return a.ToArray().Bytes(), nil
}

// Validate checks whether event is a valid NIP-42 event for given challenge and relayURL.
// The result of the validation is encoded in the ok bool.
func Validate(evt *event.T, challenge string,
	relayURL string) (pubkey string, ok bool) {

	if evt.Kind != kind.ClientAuthentication {
		return "", false
	}
	if evt.Tags.GetFirst([]string{"challenge", challenge}) == nil {
		return "", false
	}
	var expected, found *url.URL
	var e error
	expected, e = parseURL(relayURL)
	if e != nil {
		return "", false
	}
	found, e = parseURL(evt.Tags.GetFirst([]string{"relay", ""}).Value())
	if e != nil {
		return "", false
	}
	if expected.Scheme != found.Scheme ||
		expected.Host != found.Host ||
		expected.Path != found.Path {
		return "", false
	}
	now := time.Now()
	if evt.CreatedAt.Time().After(now.Add(10*time.Minute)) ||
		evt.CreatedAt.Time().Before(now.Add(-10*time.Minute)) {

		return "", false
	}
	// save for last, as it is most expensive operation
	// no need to check returned error, since ok == true implies err == nil.
	if ok, _ = evt.CheckSignature(); !ok {
		return "", false
	}
	return evt.PubKey, true
}

// helper function for Validate.
func parseURL(input string) (*url.URL, error) {
	return url.Parse(
		strings.ToLower(
			strings.TrimSuffix(input, "/"),
		),
	)
}
