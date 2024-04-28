package reqenvelope

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
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

// T is the wrapper for a query to a relay.
type T struct {
	SubscriptionID subscriptionid.T
	Filters        filters.T
}

var _ enveloper.I = &T{}

func (E *T) UnmarshalJSON(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.List[T]
func (E *T) Label() (l string) { return labels.REQ }

func (E *T) ToArray() (arr array.T) {
	arr = array.T{labels.REQ, E.SubscriptionID}
	for _, f := range E.Filters {
		arr = append(arr, f.ToObject())
	}
	return
}

func (E *T) String() (s string) { return E.ToArray().String() }

func (E *T) Bytes() (s []byte) { return E.ToArray().Bytes() }

func (E *T) MarshalJSON() ([]byte, error) { return E.Bytes(), nil }

// Unmarshal the envelope.
func (E *T) Unmarshal(buf *text.Buffer) (err error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	// log.D.F("REQ '%s'", buf.Buf[buf.Pos:])
	// Next, find the comma after the label
	if err = buf.ScanThrough(','); err != nil {
		return
	}
	var which byte
	// T can have one or no subscription IDs, if it is present we want
	// to collect it before looking for the filters.
	which, err = buf.ScanForOneOf(false, '{', '"')
	if which == '"' {
		// Next character we find will be open quotes for the subscription ID.
		if err = buf.ScanThrough('"'); err != nil {
			return
		}
		var sid []byte
		// read the string
		if sid, err = buf.ReadUntil('"'); chk.D(err) {
			return fmt.Errorf("unterminated quotes in JSON, probably truncated read: %s", err)
		}
		// log.T.F("Subscription ID: '%s'", sid)
		E.SubscriptionID = subscriptionid.T(sid)
	}
	// Next, find the comma (there must be one and at least one object brace
	// after it
	if err = buf.ScanThrough(','); err != nil {
		return
	}
	for {
		// find the opening brace of the event object, usually this is the very
		// next character, we aren't checking for valid whitespace because
		// laziness.
		if err = buf.ScanUntil('{'); chk.D(err) {
			return fmt.Errorf("event not found in event envelope: %s", err)
		}
		// now we should have an event object next. It has no embedded object so
		// it should end with a close brace. This slice will be wrapped in
		// braces and contain paired brackets, braces and quotes.
		var filterArray []byte
		if filterArray, err = buf.ReadEnclosed(); chk.D(err) {
			return
		}
		log.T.F("filter: '%s'", text.DefLimit(string(filterArray)))
		f := &filter.T{}
		if err = json.Unmarshal(filterArray, f); chk.D(err) {
			return
		}
		E.Filters = append(E.Filters, f)
		// log.D.F("remaining: '%s'", buf.Buf[buf.Pos:])
		which = 0
		if which, err = buf.ScanForOneOf(true, ',', ']'); chk.D(err) {
			return
		}
		// log.D.F("'%s'", string(which))
		if which == ']' {
			break
		}
	}
	return
}
