package countresponse

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/object"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)
var log, fails = log2.GetStd()

var (
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

type Envelope struct {
	SubscriptionID subscriptionid.T
	Count          int64
	Approximate    bool
}

func (env *Envelope) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

var _ enveloper.Enveloper = &Envelope{}

func NewCountResponseEnvelope(sid subscriptionid.T, count int64,
	approx bool) (C *Envelope) {
	C = &Envelope{
		SubscriptionID: sid,
		Count:          count,
		Approximate:    approx,
	}
	return
}
func (env *Envelope) Label() string      { return labels.EVENT }
func (env *Envelope) String() (s string) { return env.ToArray().String() }
func (env *Envelope) Bytes() (s []byte)  { return env.ToArray().Bytes() }

func (env *Envelope) ToArray() array.T {
	count := object.T{
		{Key: "count", Value: env.Count},
	}
	if env.Approximate {
		count = append(count,
			object.KV{Key: "approximate", Value: env.Approximate})
	}
	return array.T{labels.COUNT, env.SubscriptionID, count}
}

func (env *Envelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("count envelope marshal")
	return env.ToArray().Bytes(), nil
}

func (env *Envelope) Unmarshal(buf *text.Buffer) (e error) {
	log.D.Ln("ok envelope unmarshal", string(buf.Buf))
	if env == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label.
	if e = buf.ScanThrough(','); e != nil {
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
	env.SubscriptionID = subscriptionid.T(sid)
	// Next, find the comma after the subscription ID.
	if e = buf.ScanThrough(','); e != nil {
		return
	}
	var countObject []byte
	if countObject, e = buf.ReadEnclosed(); fails(e) {
		return fmt.Errorf("did not find a properly formatted JSON object for the count")
	}
	var count Count
	if e = json.Unmarshal(countObject, &count); fails(e) {
		return
	}
	env.Count = count.Count
	env.Approximate = count.Approximate
	return
}

type Count struct {
	Count       int64 `json:"count"`
	Approximate bool  `json:"approximate,omitempty"`
}
