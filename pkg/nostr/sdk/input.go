package sdk

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip5"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"
)

// InputToProfile turns any npub/nprofile/hex/nip5 input into a ProfilePointer
// (or nil).
func InputToProfile(c context.T, input string) (pp *pointers.Profile) {
	var err error
	// handle if it is a hex string
	if len(input) == 64 {
		if _, err = hex.Dec(input); !chk.E(err) {
			return &pointers.Profile{PublicKey: input}
		}
	}
	// handle nip19 codes, if that's the case
	var prefix string
	var data any
	if prefix, data, err = bech32encoding.Decode(input); log.D.Chk(err) {
	}
	var ok bool
	switch prefix {
	case "npub":
		input, ok = data.(string)
		if !ok {
			return
		}
		return &pointers.Profile{PublicKey: input}
	case "nprofile":
		pp = data.(*pointers.Profile)
		return
	}
	// handle nip5 ids, if that's the case
	if pp, err = nip5.QueryIdentifier(c, input); log.D.Chk(err) {
		return
	}
	if pp != nil {
		return
	}
	return nil
}

// InputToEventPointer turns any note/nevent/hex input into a EventPointer (or
// nil).
func InputToEventPointer(input string) (ep *pointers.Event) {
	var err error
	// handle if it is a hex string
	if len(input) == 64 {
		if _, err = hex.Dec(input); !chk.E(err) {
			return &pointers.Event{ID: eventid.T(input)}
		}
	}
	// handle nip19 codes, if that's the case
	var prefix string
	var data any
	if prefix, data, err = bech32encoding.Decode(input); log.D.Chk(err) {
		return
	}
	var ok bool
	switch prefix {
	case "note":
		if input, ok = data.(string); !ok {
			log.E.F("note pointer was not expected string")
			return
		}
		return &pointers.Event{ID: eventid.T(input)}
	case "nevent":
		if ep, ok = data.(*pointers.Event); !ok {
			log.E.F("note pointer was not event pointer")
			return
		}
		return ep
	}
	// handle nip5 ids, if that's the case (???)
	return nil
}
