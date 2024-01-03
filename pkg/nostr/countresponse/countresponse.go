package countresponse

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/object"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
	log2 "mleku.online/git/log"
)

var (
	log                    = log2.GetLogger()
	fails                  = log.D.Chk
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

type Envelope struct {
	SubscriptionID subscriptionid.T
	Count          int64
	Approximate    bool
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
func (C *Envelope) Label() labels.T    { return labels.LCount }
func (C *Envelope) String() (s string) { return C.ToArray().String() }
func (C *Envelope) Bytes() (s []byte)  { return C.ToArray().Bytes() }

func (C *Envelope) ToArray() array.T {
	count := object.T{
		{Key: "count", Value: C.Count},
	}
	if C.Approximate {
		count = append(count,
			object.KV{Key: "approximate", Value: C.Approximate})
	}
	return array.T{labels.COUNT, C.SubscriptionID, count}
}

func (C *Envelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("count envelope marshal")
	return C.ToArray().Bytes(), nil
}

func (C *Envelope) Unmarshal(buf *text.Buffer) (e error) {
	log.D.Ln("ok envelope unmarshal", string(buf.Buf))
	if C == nil {
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
	C.SubscriptionID = subscriptionid.T(sid)
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
	C.Count = count.Count
	C.Approximate = count.Approximate
	return
}

type Count struct {
	Count       int64 `json:"count"`
	Approximate bool  `json:"approximate,omitempty"`
}
