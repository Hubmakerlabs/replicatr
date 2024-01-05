package auth

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

type Challenge struct {
	Challenge string
}

var _ enveloper.Enveloper = &Challenge{}

func NewChallenge(c string) (a *Challenge) {
	return &Challenge{Challenge: c}
}

func (a *Challenge) Label() string      { return labels.AUTH }
func (a *Challenge) String() (s string) { return a.ToArray().String() }
func (a *Challenge) Bytes() (s []byte)  { return a.ToArray().Bytes() }
func (a *Challenge) ToArray() array.T {
	return array.T{labels.AUTH,
		a.Challenge}
}
func (a *Challenge) MarshalJSON() (bytes []byte, e error) {
	return a.ToArray().Bytes(), nil
}

func (a *Challenge) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

func (a *Challenge) Unmarshal(buf *text.Buffer) (e error) {
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
