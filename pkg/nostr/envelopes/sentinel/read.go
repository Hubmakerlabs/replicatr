package sentinel

import "C"
import (
	"bytes"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/authenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closedenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eoseenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/noticeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"
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
func Read(buf *text.Buffer, match string) (env enveloper.I, err error) {
	if match == "" {
		err = log.E.Err("cannot read envelope without a label match")
		return
	}
	// save the position before the comma for the Unmarshal processing.
	pos := buf.Pos
	// For most labels there is only one expected type, so we just return an
	// empty, initialized envelope struct.
	switch match {
	case labels.EVENT:
		env = &eventenvelope.T{}
	case labels.OK:
		env = &okenvelope.T{}
	case labels.NOTICE:
		env = &noticeenvelope.T{}
	case labels.EOSE:
		env = &eoseenvelope.T{}
	case labels.CLOSE:
		env = &closeenvelope.T{}
	case labels.CLOSED:
		env = &closedenvelope.T{}
	case labels.REQ:
		env = &reqenvelope.T{}
	case labels.COUNT:
		// this has two subtypes, a request and a response, the request is
		// basically like a req envelope but only wants a count response
		//
		// Both types have a subscription ID so we ensure we have a string
		// followed by a comma, the reading of it will be repeated for each
		// Unmarshal call to accommodate the more efficient, new
		// ProcessEnvelope, this function is for the UnmarshalJSON and
		// ParseEnvelope, which are not used there.
		//
		// Next, find the comma after the label.
		if err = buf.ScanThrough(','); chk.E(err) {
			return
		}
		// Next character we find will be open quotes for the subscription ID.
		if err = buf.ScanThrough('"'); chk.E(err) {
			return
		}
		var sid []byte
		// read the string
		if sid, err = buf.ReadUntil('"'); chk.E(err) {
			err = log.E.Err("unterminated quotes in JSON, probably truncated read: %s", err)
			log.D.Ln(err)
			return
		}
		s := subscriptionid.T(sid)
		// next we need to determine whether this is a request or response
		if err = buf.ScanUntil('{'); chk.E(err) {
			err = log.E.Err("event not found in event envelope")
			log.D.Ln(err)
			return
		}
		// as it is the simplest thing to look for, we search for a match on the
		// count response, which has only two fields, "count" and "approximate".
		if err = buf.ScanThrough('"'); chk.E(err) {
			return
		}
		var bb []byte
		if bb, err = buf.ReadUntil('"'); chk.E(err) {
			err = log.E.Err("unknown object in count envelope : %s '%s'",
				err, buf.String())
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
				ID: s,
			}
		}
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
			ID: s,
		}
	case labels.AUTH:
		// this has two subtypes, a request and a response, but backwards, the
		// challenge is from a relay, and the response is from a client
		// Next, find the comma after the label
		if err = buf.ScanThrough(','); chk.E(err) {
			return
		}
		var which byte
		if which, err = buf.ScanForOneOf(false, '{', '"'); chk.E(err) {
			return
		}
		switch which {
		case '"':
			env = &authenvelope.Challenge{}
		case '{':
			env = &authenvelope.Response{}
		default:
			err = log.E.Err("auth envelope malformed: '%s'", buf.String())
			return
		}
	default:
		// this should not happen so it is an error
		err = log.E.Err("unable to match envelope '%s': '%s'", match, buf)
		panic("tracing")
		return
	}
	buf.Pos = pos
	err = env.Unmarshal(buf)
	return
}
