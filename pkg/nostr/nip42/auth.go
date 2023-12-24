package nip42

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/nostric/replicatr/pkg/nostr/kind"
	"github.com/nostric/replicatr/pkg/nostr/nip1"
	"github.com/nostric/replicatr/pkg/nostr/tags"
	"github.com/nostric/replicatr/pkg/wire/array"
	"github.com/nostric/replicatr/pkg/wire/text"
	log2 "mleku.online/git/log"
)

var (
	log                    = log2.GetLogger()
	fails                  = log.D.Chk
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

const LAuth nip1.Label = 7
const AUTH = "AUTH"

func init() {
	// add this label to the nip1 envelope label map
	nip1.Labels[LAuth] = []byte(AUTH)
}

type AuthChallengeEnvelope struct {
	Challenge string
}

func (a *AuthChallengeEnvelope) Label() nip1.Label {
	return LAuth
}

func (a *AuthChallengeEnvelope) Unmarshal(buf *text.Buffer) (e error) {
	log.D.Ln("ok envelope unmarshal", string(buf.Buf))
	if a == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label
	if e = buf.ScanThrough(','); e != nil {
		return
	}
	// next comes the challenge string
	if e = buf.ScanThrough('"'); e != nil {
		return
	}
	var challengeString []byte
	if challengeString, e = buf.ReadUntil('"'); fails(e) {
		return fmt.Errorf("did not find challenge string in auth challenge envelope")
	}
	a.Challenge = string(text.UnescapeByteString(challengeString))
	// Scan for the proper envelope ending.
	if e = buf.ScanThrough(']'); e != nil {
		log.D.Ln("envelope unterminated but all fields found")
	}
	return
}

func (a *AuthChallengeEnvelope) ToArray() array.T {
	return array.T{AUTH, a.Challenge}
}

func (a *AuthChallengeEnvelope) String() (s string) {
	return a.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (a *AuthChallengeEnvelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("auth challenge envelope marshal")
	return a.ToArray().Bytes(), nil
}

type AuthResponseEnvelope struct {
	*nip1.Event
}

// New creates an AuthResponseEnvelope response from an AuthChallengeEnvelope.
//
// The caller must sign the embedded event before sending it back to
// authenticate.
func New(ac *AuthChallengeEnvelope, relay string) (ae *AuthResponseEnvelope) {
	ae = &AuthResponseEnvelope{
		&nip1.Event{
			Kind: kind.ClientAuthentication,
			Tags: tags.T{
				{"relay", relay},
				{"challenge", ac.Challenge},
			},
		},
	}
	return
}

func (a *AuthResponseEnvelope) Label() nip1.Label { return LAuth }

func (a *AuthResponseEnvelope) Unmarshal(buf *text.Buffer) (e error) {
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
	a.Event = &nip1.Event{}
	if e = json.Unmarshal(eventObj, a.Event); fails(e) {
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

func (a *AuthResponseEnvelope) ToArray() array.T {
	return array.T{AUTH, a.Event.ToObject()}
}

func (a *AuthResponseEnvelope) String() (s string) {
	return a.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (a *AuthResponseEnvelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("auth envelope marshal")
	return a.ToArray().Bytes(), nil
}
