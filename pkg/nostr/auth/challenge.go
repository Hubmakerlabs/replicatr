package auth

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

type ChallengeEnvelope struct {
	Challenge string
}

func NewChallenge(c string) (a *ChallengeEnvelope) {
	return &ChallengeEnvelope{Challenge: c}
}

func (a *ChallengeEnvelope) Label() labels.T    { return labels.LAuth }
func (a *ChallengeEnvelope) String() (s string) { return a.ToArray().String() }
func (a *ChallengeEnvelope) Bytes() (s []byte)  { return a.ToArray().Bytes() }

func (a *ChallengeEnvelope) ToArray() array.T {
	return array.T{labels.List[labels.LAuth], a.Challenge}
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (a *ChallengeEnvelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("auth challenge envelope marshal")
	return a.ToArray().Bytes(), nil
}

func (a *ChallengeEnvelope) Unmarshal(buf *text.Buffer) (e error) {
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
