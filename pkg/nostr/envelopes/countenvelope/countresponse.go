package countenvelope

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/enveloper"
	l "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/object"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

type Response struct {
	SubscriptionID subscriptionid.T
	Count          int64
	Approximate    bool
}

var _ enveloper.I = &Response{}

func (env *Response) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

func New(sid subscriptionid.T, count int64, approx bool) (C *Response) {
	C = &Response{
		SubscriptionID: sid,
		Count:          count,
		Approximate:    approx,
	}
	return
}
func (env *Response) Label() string { return l.EVENT }

func (env *Response) ToArray() array.T {
	count := object.T{
		{Key: "count", Value: env.Count},
	}
	if env.Approximate {
		count = append(count,
			object.KV{Key: "approximate", Value: env.Approximate})
	}
	return array.T{l.COUNT, env.SubscriptionID, count}
}

func (env *Response) String() (s string) { return env.ToArray().String() }

func (env *Response) Bytes() (s []byte) { return env.ToArray().Bytes() }

func (env *Response) MarshalJSON() ([]byte, error) { return env.Bytes(), nil }

func (env *Response) Unmarshal(buf *text.Buffer) (e error) {
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
	if sid, e = buf.ReadUntil('"'); log.Fail(e) {
		return fmt.Errorf("unterminated quotes in JSON, " +
			"probably truncated read")
	}
	env.SubscriptionID = subscriptionid.T(sid)
	// Next, find the comma after the subscription ID.
	if e = buf.ScanThrough(','); e != nil {
		return
	}
	var countObject []byte
	if countObject, e = buf.ReadEnclosed(); log.Fail(e) {
		return fmt.Errorf("did not find a properly formatted JSON " +
			"object for the count")
	}
	var count Count
	if e = json.Unmarshal(countObject, &count); log.Fail(e) {
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
