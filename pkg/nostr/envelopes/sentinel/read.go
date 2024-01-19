package sentinel

import "C"
import (
	"bytes"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/authenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closedenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eoseenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/noticeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/text"
)

var countString = []byte("count")
var approximateString = []byte("approximate")

// Read performs a second stage process after Identify that allocates a proper
// data structure for the Unmarshal function to populate with the remainder of
// the buffer.
//
// This function exists because of the count and auth envelopes that have one
// label but two distinct structures - they are different by the role of the
// sender but need extra scanning to distinguish between them.
func Read(buf *text.Buffer, match labels.T) (env enveloper.I, e error) {
	// For most labels there is only one expected type, so we just return an
	// empty, initialized envelope struct.
	switch match {
	case labels.LEvent:
		env = &eventenvelope.T{}
	case labels.LOK:
		env = &okenvelope.T{}
	case labels.LNotice:
		env = &noticeenvelope.T{}
	case labels.LEOSE:
		env = &eoseenvelope.T{}
	case labels.LClose:
		env = &closeenvelope.T{}
	case labels.LClosed:
		env = &closedenvelope.T{}
	case labels.LReq:
		env = &reqenvelope.T{}
	case labels.LCount:
		// this has two subtypes, a request and a response, the request is
		// basically like a req envelope but only wants a count response
		//
		// Both types have a subscription ID so we ensure we have a string
		// followed by a comma, the reading of it will be repeated for each
		// Unmarshal call to accommodate the more efficient, new
		// ProcessEnvelope, this function is for the UnmarshalJSON and
		// ParseEnvelope, which are not used there.
		//
		// save the position before the comma for the Unmarshal processing.
		pos := buf.Pos
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
			e = fmt.Errorf("unterminated quotes in JSON, probably truncated read")
			log.D.Ln(e)
			return
		}
		s := subscriptionid.T(sid)
		// next we need to determine whether this is a request or response
		if e = buf.ScanUntil('{'); e != nil {
			e = fmt.Errorf("event not found in event envelope")
			log.D.Ln(e)
			return
		}
		// as it is the simplest thing to look for, we search for a match on the
		// count response, which has only two fields, "count" and "approximate".
		if e = buf.ScanThrough('"'); e != nil {
			return
		}
		var bb []byte
		if bb, e = buf.ReadUntil('"'); log.Fail(e) {
			e = fmt.Errorf("unknown object in count envelope '%s'",
				buf.String())
			return
		}
		// we should now have a string to compare
		//
		// we assume here that json encoding at least respects case for keys,
		// generally it does and cases for keys are clearly specified in the
		// NIPs. (it doesn't seem like it should ever be necessary to use a
		// ToLower function on the string but just noting that this may not be
		// true).
		if bytes.Compare(bb, countString) == 0 ||
			bytes.Compare(bb, approximateString) == 0 {
			// we found a valid count response object, probably, the rest of the
			// object should be a count response.
			env = &countenvelope.Response{
				SubscriptionID: s,
			}
		} else {
			// we only check if it matches one of the two possible count
			// response key strings, as this is the smaller operation than
			// checking if the following object is a filter, thus, it is assumed
			// here that the object is a filter as it doesn't contain keys from
			// a count response
			//
			// a COUNT envelope could have many filters but it doesn't matter
			// because we are only concerned with correctly identifying whether
			// this is a count response or request
			env = &countenvelope.Request{
				SubscriptionID: s,
			}
		}
		// restore the position to prior to the first filter or the response
		// count object
		buf.Pos = pos
	case labels.LAuth:
		// save the position before the comma for the auth.Response Unmarshal
		pos := buf.Pos
		// this has two subtypes, a request and a response, but backwards, the
		// challenge is from a relay, and the response is from a client
		// Next, find the comma after the label
		if e = buf.ScanThrough(','); e != nil {
			return
		}
		var which byte
		if which, e = buf.ScanForOneOf(false, '{', '"'); log.Fail(e) {
			return
		}
		switch which {
		case '"':
			env = &authenvelope.Response{}
			buf.Pos = pos
		case '{':
			env = &authenvelope.Response{}
			buf.Pos = pos
		default:
			e = fmt.Errorf("auth envelope malformed: '%s'", buf.String())
			log.D.Ln(e)
		}
		return
	default:
	}
	// this should not happen so it is an error
	e = fmt.Errorf("unable to match envelope type number %d '%s': '%s'",
		match, labels.List[match], buf.String())
	return
}
