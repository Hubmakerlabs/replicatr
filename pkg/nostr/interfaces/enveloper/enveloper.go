package enveloper

import (
	"encoding/json"
	"fmt"

	"mleku.dev/git/nostr/interfaces/arrayer"
	"mleku.dev/git/nostr/interfaces/buffer"
	"mleku.dev/git/nostr/interfaces/byter"
	"mleku.dev/git/nostr/interfaces/labeler"
)

// I interface for envelopes
//
// Note that the Unmarshal function is not UnmarshalJSON for a specific reason -
// it is impossible to implement a typed JSON unmarshaler in Go for an array
// type because it must by definition have a sentinel field which in the case of
// nostr is the Label. Objects have a defined collection of recognised labels
// and with omitempty marking the mandatory ones, acting as a "kind" of
// sentinel.
type I interface {
	labeler.I
	fmt.Stringer
	byter.I
	json.Marshaler
	buffer.Unmarshaler
	arrayer.I
}
