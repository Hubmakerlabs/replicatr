package app

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
)

// Chat implements the control interface, intercepting kind 4 encrypted direct
// messages and processing them if they are for the relay's pubkey
func (rl *Relay) Chat(c context.T, ev *event.T) {
	if ev.Kind != kind.EncryptedDirectMessage && ev.PubKey != rl.RelayPubHex {
		log.T.Ln("direct message not for relay chat")
		return
	}
	log.T.Ln(rl.RelayPubHex, "receiving message via DM")
}
