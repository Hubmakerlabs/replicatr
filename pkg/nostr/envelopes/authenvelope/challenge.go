package authenvelope

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/interfaces/enveloper"
	l "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

type Challenge struct {
	Challenge string
}

var _ enveloper.I = &Challenge{}

func (a *Challenge) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

func NewChallenge(c string) (a *Challenge) {
	return &Challenge{Challenge: c}
}

func (a *Challenge) Label() string { return l.AUTH }

func (a *Challenge) String() string { return a.ToArray().String() }

func (a *Challenge) Bytes() []byte { return a.ToArray().Bytes() }

func (a *Challenge) ToArray() array.T { return array.T{l.AUTH, a.Challenge} }

func (a *Challenge) MarshalJSON() ([]byte, error) { return a.Bytes(), nil }

func (a *Challenge) Unmarshal(buf *text.Buffer) (e error) {
	log.D.Ln("auth challenge envelope unmarshal", string(buf.Buf))
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
	if challengeString, e = buf.ReadUntil('"'); log.Fail(e) {
		return fmt.Errorf("did not find challenge string in auth challenge envelope")
	}
	a.Challenge = string(text.UnescapeByteString(challengeString))
	// Scan for the proper envelope ending.
	if e = buf.ScanThrough(']'); e != nil {
		log.D.Ln("envelope unterminated but all fields found")
	}
	return
}
