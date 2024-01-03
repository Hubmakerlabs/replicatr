package nip45

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
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

const LCount labels.T = 9
const COUNT = "COUNT"

func init() {
	// add this label to the nip1 envelope label map
	labels.List[LCount] = []byte(COUNT)
}

type CountRequestEnvelope struct {
	SubscriptionID subscriptionid.T
	filters.T
}

func (C *CountRequestEnvelope) Label() labels.T    { return LCount }
func (C *CountRequestEnvelope) String() (s string) { return C.ToArray().String() }
func (C *CountRequestEnvelope) Bytes() (s []byte)  { return C.ToArray().Bytes() }

func (C *CountRequestEnvelope) ToArray() array.T {
	return array.T{COUNT, C.SubscriptionID, C.T}
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (C *CountRequestEnvelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("count request envelope marshal")
	return C.ToArray().Bytes(), nil
}

func (C *CountRequestEnvelope) Unmarshal(buf *text.Buffer) (e error) {
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
	// find the opening brace of the first or only filter object.
	if e = buf.ScanUntil('{'); e != nil {
		return fmt.Errorf("event not found in event envelope")
	}
	// T in the count envelope are variadic, there can be more than one,
	// with subsequent items separated by a comma, so we read them in in a loop,
	// breaking when we don't find a comma after.
	for {
		var filterArray []byte
		if filterArray, e = buf.ReadEnclosed(); fails(e) {
			return
		}
		f := &filter.T{}
		if e = json.Unmarshal(filterArray, &f); fails(e) {
			return
		}
		C.T = append(C.T, f)
		cur := buf.Pos
		// Next, find the comma after filter.
		if e = buf.ScanThrough(','); e != nil {
			// we didn't find one, so break the loop.
			buf.Pos = cur
			break
		}
	}
	// If we found at least one filter, there is no error, the io.EOF is
	// expected at any point after at least one filter.
	if len(C.T) > 0 {
		e = nil
	}
	// // Technically we maybe should read ahead further to make sure the JSON
	// // closes correctly. Not going to abort because of this.
	// //
	// TODO: this is a waste of time really, the rest of the buffer will be
	//  discarded anyway as no more content is expected
	// if e = buf.ScanUntil(']'); e != nil {
	// 	return fmt.Errorf("malformed JSON, no closing bracket on array")
	// }
	// whatever remains doesn't matter as the envelope has fully unmarshaled.
	return
}

type CountResponseEnvelope struct {
	SubscriptionID subscriptionid.T
	Count          int64
	Approximate    bool
}

var _ enveloper.Enveloper = &CountResponseEnvelope{}

func NewCountResponseEnvelope(sid subscriptionid.T, count int64,
	approx bool) (C *CountResponseEnvelope) {
	C = &CountResponseEnvelope{
		SubscriptionID: sid,
		Count:          count,
		Approximate:    approx,
	}
	return
}
func (C *CountResponseEnvelope) Label() labels.T    { return LCount }
func (C *CountResponseEnvelope) String() (s string) { return C.ToArray().String() }
func (C *CountResponseEnvelope) Bytes() (s []byte)  { return C.ToArray().Bytes() }

func (C *CountResponseEnvelope) ToArray() array.T {
	count := object.T{
		{Key: "count", Value: C.Count},
	}
	if C.Approximate {
		count = append(count,
			object.KV{Key: "approximate", Value: C.Approximate})
	}
	return array.T{COUNT, C.SubscriptionID, count}
}

func (C *CountResponseEnvelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("count envelope marshal")
	return C.ToArray().Bytes(), nil
}

func (C *CountResponseEnvelope) Unmarshal(buf *text.Buffer) (e error) {
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
