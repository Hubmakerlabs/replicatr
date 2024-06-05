package countenvelope

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
	"mleku.net/slog"
)

var log, chk = slog.New(os.Stderr)

type Request struct {
	ID      subscriptionid.T
	Filters filters.T
}

var _ enveloper.I = &Request{}

func (C *Request) Label() string { return labels.COUNT }
func (C *Request) ToArray() array.T {
	return array.T{labels.COUNT,
		C.ID, C.Filters}
}
func (C *Request) String() string               { return C.ToArray().String() }
func (C *Request) Bytes() []byte                { return C.ToArray().Bytes() }
func (C *Request) MarshalJSON() ([]byte, error) { return C.Bytes(), nil }

func (C *Request) Unmarshal(buf *text.Buffer) (err error) {
	log.D.Ln("ok envelope unmarshal", string(buf.Buf))
	if C == nil {
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
		return fmt.Errorf("unterminated quotes in JSON, probably truncated read: %s",
			err)
	}
	C.ID = subscriptionid.T(sid)
	// find the opening brace of the first or only filter object.
	if err = buf.ScanUntil('{'); chk.D(err) {
		return fmt.Errorf("event not found in event envelope: %s", err)
	}
	// T in the count envelope are variadic, there can be more than one, with
	// subsequent items separated by a comma, so we read them in in a loop, breaking
	// when we don't find a comma after.
	for {
		var filterArray []byte
		if filterArray, err = buf.ReadEnclosed(); chk.D(err) {
			return
		}
		f := &filter.T{}
		if err = json.Unmarshal(filterArray, &f); chk.D(err) {
			return
		}
		C.Filters = append(C.Filters, f)
		cur := buf.Pos
		// Next, find the comma after filter.
		if err = buf.ScanThrough(','); err != nil {
			// we didn't find one, so break the loop.
			buf.Pos = cur
			break
		}
	}
	// If we found at least one filter, there is no error, the io.EOF is expected at
	// any point after at least one filter.
	if len(C.Filters) > 0 {
		err = nil
	}
	// Technically we maybe should read ahead further to make sure the JSON closes
	// correctly. Not going to abort because of this. Whatever remains doesn't
	// matter as the envelope has fully unmarshalled.
	return
}
