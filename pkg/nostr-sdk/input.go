package sdk

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip05"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"
)

// InputToProfile turns any npub/nprofile/hex/nip05 input into a ProfilePointer (or nil).
func InputToProfile(c context.T, input string) *pointers.Profile {
	// handle if it is a hex string
	if len(input) == 64 {
		if _, e := hex.Dec(input); e == nil {
			return &pointers.Profile{PublicKey: input}
		}
	}

	// handle nip19 codes, if that's the case
	prefix, data, _ := bech32encoding.Decode(input)
	switch prefix {
	case "npub":
		input = data.(string)
		return &pointers.Profile{PublicKey: input}
	case "nprofile":
		pp := data.(pointers.Profile)
		return &pp
	}

	// handle nip05 ids, if that's the case
	pp, _ := nip05.QueryIdentifier(c, input)
	if pp != nil {
		return pp
	}

	return nil
}

// InputToEventPointer turns any note/nevent/hex input into a EventPointer (or nil).
func InputToEventPointer(input string) *pointers.Event {
	// handle if it is a hex string
	if len(input) == 64 {
		if _, e := hex.Dec(input); e == nil {
			return &pointers.Event{ID: eventid.T(input)}
		}
	}

	// handle nip19 codes, if that's the case
	prefix, data, _ := bech32encoding.Decode(input)
	switch prefix {
	case "note":
		input = data.(string)
		return &pointers.Event{ID: eventid.T(input)}
	case "nevent":
		ep := data.(pointers.Event)
		return &ep
	}

	// handle nip05 ids, if that's the case
	return nil
}
