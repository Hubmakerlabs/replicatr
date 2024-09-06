package authenvelope

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
)

type Challenge struct {
	Challenge string
}

var _ enveloper.I = &Challenge{}

func NewChallenge(c string) (a *Challenge)        { return &Challenge{Challenge: c} }
func (a *Challenge) Label() string                { return labels.AUTH }
func (a *Challenge) String() string               { return a.ToArray().String() }
func (a *Challenge) Bytes() []byte                { return a.ToArray().Bytes() }
func (a *Challenge) ToArray() array.T             { return array.T{labels.AUTH, a.Challenge} }
func (a *Challenge) MarshalJSON() ([]byte, error) { return a.Bytes(), nil }

func (a *Challenge) Unmarshal(buf *text.Buffer) (err error) {
	log.D.Ln("auth challenge envelope unmarshal", string(buf.Buf))
	if a == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label
	if err = buf.ScanThrough(','); err != nil {
		return
	}
	// next comes the challenge string
	if err = buf.ScanThrough('"'); err != nil {
		return
	}
	var challengeString []byte
	if challengeString, err = buf.ReadUntil('"'); chk.D(err) {
		return fmt.Errorf("did not find challenge string in auth challenge envelope: %s", err)
	}
	a.Challenge = string(text.UnescapeByteString(challengeString))
	// Scan for the proper envelope ending.
	if err = buf.ScanThrough(']'); chk.D(err) {
		log.D.Ln("envelope unterminated but all fields found: %s", err)
	}
	return
}
