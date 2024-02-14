package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
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
func (rl *Relay) Chat(c context.T, ev *event.T) (err error) {
	if ev.Kind != kind.EncryptedDirectMessage {
		return
	}
	if !ev.Tags.ContainsAny("p", rl.RelayPubHex) {
		log.T.Ln("direct message not for relay chat", ev.PubKey, rl.RelayPubHex)
		return
	}
	go func() {
		log.I.Ln(rl.RelayPubHex, "receiving message via DM", ev.ToObject().String())
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
		ws := GetConnection(c)
		input := strings.TrimSpace(decryptedStr)
		if ws.AuthPubKey.Load() == "" {
			if strings.HasPrefix(input, "AUTH:") {
				var authed bool
				authStr := strings.Split(decryptedStr, ":")
				log.I.Ln(authStr, ws.Challenge.Load())
				if len(authStr) == 3 {
					var ts int64
					if ts, err = strconv.ParseInt(authStr[1], 10, 64); chk.E(err) {
						return
					}
					now := timestamp.Now().Time().Unix()
					log.I.Ln()
					if ts < now+15 || ts > now-15 {
						if authStr[2] == ws.Challenge.Load() {
							authed = true
							ws.AuthPubKey.Store(ev.PubKey)
						}
					}
				}
				if !authed {
					content := fmt.Sprintf("access denied")
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
					rl.BroadcastEvent(reply)
					// create a new challenge for this connection
					challenge := make([]byte, 8)
					_, err = rand.Read(challenge)
					chk.E(err)
					ws.Challenge.Store(hex.EncodeToString(challenge))
					return
				} else {
					content := fmt.Sprintf("access granted, now executing "+
						"previously entered command: '%s'", ws.Pending.Load())
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
					rl.BroadcastEvent(reply)
					// now process cached
					log.I.Ln("pending message:", ws.Pending.Load())
					go rl.command(ws.Pending.Load())
					// erase
					ws.Pending.Store("")
					return
				}
			} else {
				// store the input in the websocket state to process after
				// successful auth
				ws.Pending.Store(decryptedStr)
				content := fmt.Sprintf(
					"please reply within 15 seconds with the following text:"+
						"\n\nAUTH:%d:%v",
					timestamp.Now(), ws.Challenge.Load())
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
				rl.BroadcastEvent(reply)
				return
			}
		}
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
		rl.BroadcastEvent(reply)
	}()

	return
}

func (rl *Relay) command(cmd string) {
	log.D.Ln("executing command '%s'", cmd)
}
