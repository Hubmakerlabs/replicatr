package sdk

import (
	"context"
	"encoding/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip5"
)

// InputToProfile turns any npub/nprofile/hex/nip5 input into a ProfilePointer
// (or nil).
func InputToProfile(ctx context.Context, input string) (pp *pointers.Profile) {
	// handle if it is a hex string
	if len(input) == 64 {
		if _, err := hex.DecodeString(input); err == nil {
			return &pointers.Profile{PublicKey: input}
		}
	}
	// handle nip19 codes, if that's the case
	prefix, data, e := nip19.Decode(input)
	log.D.Chk(e)
	switch prefix {
	case "npub":
		input = data.(string)
		return &pointers.Profile{PublicKey: input}
	case "nprofile":
		pp := data.(pointers.Profile)
		return &pp
	}
	// handle nip5 ids, if that's the case
	pp, e = nip5.QueryIdentifier(ctx, input)
	log.D.Chk(e)
	if pp != nil {
		return pp
	}
	return nil
}

// InputToEventPointer turns any note/nevent/hex input into a EventPointer (or
// nil).
func InputToEventPointer(input string) (ep *pointers.Event) {
	// handle if it is a hex string
	if len(input) == 64 {
		if _, err := hex.DecodeString(input); err == nil {
			return &pointers.Event{ID: nip1.EventID(input)}
		}
	}
	// handle nip19 codes, if that's the case
	prefix, data, e := nip19.Decode(input)
	log.D.Chk(e)
	switch prefix {
	case "note":
		input = data.(string)
		return &pointers.Event{ID: nip1.EventID(input)}
	case "nevent":
		*ep = data.(pointers.Event)
		return ep
	}
	// handle nip5 ids, if that's the case (???)
	return nil
}
