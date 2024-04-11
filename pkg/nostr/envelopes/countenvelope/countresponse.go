package countenvelope

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/object"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
)

type Response struct {
	ID          subscriptionid.T
	Count       int
	Approximate bool
}

var _ enveloper.I = &Response{}

func (env *Response) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

func New(sid subscriptionid.T, count int, approx bool) (C *Response) {
	C = &Response{
		ID:          sid,
		Count:       count,
		Approximate: approx,
	}
	return
}
func (env *Response) Label() string { return labels.COUNT }

func (env *Response) ToArray() array.T {
	count := object.T{
		{Key: "count", Value: env.Count},
	}
	if env.Approximate {
		count = append(count,
			object.KV{Key: "approximate", Value: env.Approximate})
	}
	return array.T{labels.COUNT, env.ID, count}
}

func (env *Response) String() (s string) { return env.ToArray().String() }

func (env *Response) Bytes() (s []byte) { return env.ToArray().Bytes() }

func (env *Response) MarshalJSON() ([]byte, error) { return env.Bytes(), nil }

func (env *Response) Unmarshal(buf *text.Buffer) (err error) {
	log.D.Ln("ok envelope unmarshal", string(buf.Buf))
	if env == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// Next, find the comma after the label.
	if err = buf.ScanThrough(','); err != nil {
		return
	}
	// Next character we find will be open quotes for the subscription ID.
	if err = buf.ScanThrough('"'); err != nil {
		return
	}
	var sid []byte
	// read the string
	if sid, err = buf.ReadUntil('"'); chk.D(err) {
		return fmt.Errorf("unterminated quotes in JSON, "+
			"probably truncated read: %s", err)
	}
	env.ID = subscriptionid.T(sid)
	// Next, find the comma after the subscription ID.
	if err = buf.ScanThrough(','); err != nil {
		return
	}
	var countObject []byte
	if countObject, err = buf.ReadEnclosed(); chk.D(err) {
		return fmt.Errorf("did not find a properly formatted JSON "+
			"object for the count: %s", err)
	}
	var count Count
	if err = json.Unmarshal(countObject, &count); chk.D(err) {
		return
	}
	env.Count = count.Count
	env.Approximate = count.Approximate
	return
}

type Count struct {
	Count       int  `json:"count"`
	Approximate bool `json:"approximate,omitempty"`
}
