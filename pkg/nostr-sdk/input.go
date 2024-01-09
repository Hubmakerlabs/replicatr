package sdk

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip05"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/pointers"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
)

// InputToProfile turns any npub/nprofile/hex/nip05 input into a ProfilePointer (or nil).
func InputToProfile(ctx context.T, input string) *pointers.ProfilePointer {
	// handle if it is a hex string
	if len(input) == 64 {
		if _, e := hex.Dec(input); e == nil {
			return &pointers.ProfilePointer{PublicKey: input}
		}
	}

	// handle nip19 codes, if that's the case
	prefix, data, _ := nip19.Decode(input)
	switch prefix {
	case "npub":
		input = data.(string)
		return &pointers.ProfilePointer{PublicKey: input}
	case "nprofile":
		pp := data.(pointers.ProfilePointer)
		return &pp
	}

	// handle nip05 ids, if that's the case
	pp, _ := nip05.QueryIdentifier(ctx, input)
	if pp != nil {
		return pp
	}

	return nil
}

// InputToEventPointer turns any note/nevent/hex input into a EventPointer (or nil).
func InputToEventPointer(input string) *pointers.EventPointer {
	// handle if it is a hex string
	if len(input) == 64 {
		if _, e := hex.Dec(input); e == nil {
			return &pointers.EventPointer{ID: input}
		}
	}

	// handle nip19 codes, if that's the case
	prefix, data, _ := nip19.Decode(input)
	switch prefix {
	case "note":
		input = data.(string)
		return &pointers.EventPointer{ID: input}
	case "nevent":
		ep := data.(pointers.EventPointer)
		return &ep
	}

	// handle nip05 ids, if that's the case
	return nil
}
