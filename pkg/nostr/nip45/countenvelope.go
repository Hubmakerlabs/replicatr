package nip45

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
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

const LCount nip1.Label = 9
const COUNT = "COUNT"

func init() {
	// add this label to the nip1 envelope label map
	nip1.Labels[LCount] = []byte(COUNT)
}

type CountRequestEnvelope struct {
	SubscriptionID nip1.SubscriptionID
	nip1.Filters
}

func (C *CountRequestEnvelope) Label() nip1.Label  { return LCount }
func (C *CountRequestEnvelope) String() (s string) { return C.ToArray().String() }
func (C *CountRequestEnvelope) Bytes() (s []byte)  { return C.ToArray().Bytes() }

func (C *CountRequestEnvelope) ToArray() array.T {
	return array.T{COUNT, C.SubscriptionID, C.Filters}
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (C *CountRequestEnvelope) MarshalJSON() (bytes []byte, e error) {
	// log.D.F("count envelope marshal")
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
	C.SubscriptionID = nip1.SubscriptionID(sid)
	// find the opening brace of the first or only filter object.
	if e = buf.ScanUntil('{'); e != nil {
		return fmt.Errorf("event not found in event envelope")
	}
	// Filters in the count envelope are variadic, there can be more than one,
	// with subsequent items separated by a comma, so we read them in in a loop,
	// breaking when we don't find a comma after.
	for {
		var filterArray []byte
		if filterArray, e = buf.ReadEnclosed(); fails(e) {
			return
		}
		var f nip1.Filter
		if e = json.Unmarshal(filterArray, &f); fails(e) {
			return
		}
		C.Filters = append(C.Filters, f)
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
	if len(C.Filters) > 0 {
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
	SubscriptionID nip1.SubscriptionID
	Count          int64
	Approximate    bool
}

func NewCountResponseEnvelope(sid nip1.SubscriptionID, count int64,
	approx bool) (C *CountResponseEnvelope) {
	C = &CountResponseEnvelope{
		SubscriptionID: sid,
		Count:          count,
		Approximate:    approx,
	}
	return
}
func (C *CountResponseEnvelope) Label() nip1.Label  { return LCount }
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
	C.SubscriptionID = nip1.SubscriptionID(sid)
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
