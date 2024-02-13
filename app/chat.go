package app

import (
	"fmt"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip4"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

// Chat implements the control interface, intercepting kind 4 encrypted direct
// messages and processing them if they are for the relay's pubkey
func (rl *Relay) Chat(c context.T, ev *event.T) {
	if ev.Kind != kind.EncryptedDirectMessage {
		return
	}
	if !ev.Tags.ContainsAny("p", rl.RelayPubHex) {
		log.T.Ln("direct message not for relay chat", ev.PubKey, rl.RelayPubHex)
		return
	}
	log.I.Ln(rl.RelayPubHex, "receiving message via DM", ev.ToObject().String())
	var err error
	var secret, decrypted []byte
	if secret, err = nip4.ComputeSharedSecret(ev.PubKey,
		rl.Config.SecKey); chk.E(err) {
		return
	}
	if decrypted, err = nip4.Decrypt(ev.Content, secret); chk.E(err) {
		return
	}
	decryptedStr := string(decrypted)
	log.I.F("decrypted message: '%s'", decryptedStr)
	split := strings.Split(decryptedStr, " ")
	content := fmt.Sprintf("array of space separated fields of message: %v",
		split)
	reply := &event.T{
		CreatedAt: timestamp.Now() + 2,
		Kind:      kind.EncryptedDirectMessage,
		Tags:      tags.T{{"p", ev.PubKey}, {"e", ev.ID.String()}},
	}
	if reply.Content, err = nip4.Encrypt(content, secret); chk.E(err) {
		return
	}
	if err = reply.Sign(rl.Config.SecKey); chk.E(err) {
		return
	}
	log.I.Ln("reply", reply.ToObject().String())
	for i, store := range rl.StoreEvent {
		log.T.Ln("running event store function", i)
		if err = store(c, reply); chk.T(err) {
			return
		}
	}
	rl.BroadcastEvent(reply)
}
