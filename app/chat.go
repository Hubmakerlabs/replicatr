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
	"github.com/davecgh/go-spew/spew"
)

// DecryptDM decrypts a DM, kind 4, 1059 or 1060
func DecryptDM(ev *event.T, meSec, youPub string) (decryptedStr string, err error) {
	switch ev.Kind {
	case kind.EncryptedDirectMessage:
		var secret, decrypted []byte
		if secret, err = nip4.ComputeSharedSecret(meSec, youPub); chk.E(err) {
			return
		}
		if decrypted, err = nip4.Decrypt(ev.Content, secret); chk.E(err) {
			return
		}
		decryptedStr = string(decrypted)
	case kind.GiftWrap:
	case kind.GiftWrapWithKind4:
	}
	return
}

// EncryptDM encrypts a DM, kind 4, 1059 or 1060
func EncryptDM(ev *event.T, meSec, youPub string) (evo *event.T, err error) {
	var secret []byte
	switch ev.Kind {
	case kind.EncryptedDirectMessage:
		if secret, err = nip4.ComputeSharedSecret(meSec, youPub); chk.E(err) {
			return
		}
		if ev.Content, err = nip4.Encrypt(ev.Content, secret); chk.E(err) {
			return
		}
		if err = ev.Sign(meSec); chk.E(err) {
			return
		}
	case kind.GiftWrap:
	case kind.GiftWrapWithKind4:
	}
	evo = ev
	return
}

// MakeReply creates an appropriate reply event from a provided event that is
// being replied to (not quoting, just the right tags, timestamps and kind).
func MakeReply(ev *event.T, content string) (evo *event.T) {
	created := ev.CreatedAt + 2
	now := timestamp.Now()
	if created < now {
		created = now
	}
	evo = &event.T{
		CreatedAt: created,
		Kind:      ev.Kind,
		Tags:      tags.T{{"p", ev.PubKey}, {"e", ev.ID.String()}},
		Content:   content,
	}
	return
}

// Chat implements the control interface, intercepting kind 4 encrypted direct
// messages and processing them if they are for the relay's pubkey
func (rl *Relay) Chat(c context.T, ev *event.T) (err error) {
	if ev.Kind != kind.EncryptedDirectMessage {
		kind.MapMx.Lock()
		log.I.Ln("not chat event", ev.Kind, kind.GetString(ev.Kind))
		kind.MapMx.Unlock()
		return
	}
	if !ev.Tags.ContainsAny("p", rl.RelayPubHex) && ev.PubKey != rl.RelayPubHex {
		log.T.Ln("direct message not for relay chat", ev.PubKey, rl.RelayPubHex)
		return
	}
	meSec, youPub := rl.Config.SecKey, ev.PubKey
	log.I.Ln(rl.RelayPubHex, "receiving message via DM", ev.ToObject().String())
	var decryptedStr string
	decryptedStr, err = DecryptDM(ev, meSec, youPub)
	log.I.F("decrypted message: '%s'", decryptedStr)
	ws := GetConnection(c)
	decryptedStr = strings.TrimSpace(decryptedStr)
	var reply *event.T
	if ws.AuthPubKey.Load() == "" {
		if strings.HasPrefix(decryptedStr, "AUTH_") {
			var authed bool
			authStr := strings.Split(decryptedStr, "_")
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
				reply = MakeReply(ev, fmt.Sprintf("access denied"))
				if reply, err = EncryptDM(reply, meSec, youPub); chk.E(err) {
					return
				}
				log.I.Ln("reply", reply.ToObject().String())
				rl.BroadcastEvent(reply)
				// create a new challenge for this connection
				challenge := make([]byte, 8)
				if _, err = rand.Read(challenge); chk.E(err) {
					// i never know what to do for this case, panic? usually
					// just ignore, it should never happen
				}
				ws.Challenge.Store(hex.EncodeToString(challenge))
				return
			} else {
				// reply = MakeReply(ev,
				// 	fmt.Sprintf("access granted, now executing previously"+
				// 		" entered command: '%v'", ws.Pending.Load()))
				// if reply, err = EncryptDM(reply, meSec, youPub); chk.E(err) {
				// 	return
				// }
				// log.I.Ln("reply", reply.ToObject().String())
				// rl.BroadcastEvent(reply)
				// now process cached
				log.I.Ln("pending message:", ws.Pending.Load())
				cmd := ws.Pending.Load().(string)
				// erase
				ws.Pending.Store("")
				chk.E(rl.command(ev, cmd))
				return
			}
		} else {
			// store the input in the websocket state to process after
			// successful auth
			ws.Pending.Store(decryptedStr)
			content := fmt.Sprintf(`
received command

%s

to authorise executing this command, please reply within 15 seconds with the following text:

AUTH_%d_%v

after this you will not have to do this again unless there is a long idle, disconnect or you refresh your session

note that if you have NIP-42 enabled in the client and you are already authorised this notice will not appear
`, decryptedStr, timestamp.Now(), ws.Challenge.Load())
			reply = MakeReply(ev, content)
			if reply, err = EncryptDM(reply, meSec, youPub); chk.E(err) {
				return
			}
			log.I.Ln("reply", reply.ToObject().String())
			rl.BroadcastEvent(reply)
			return
		}
	} else {
		if err = rl.command(ev, decryptedStr); chk.E(err) {
			return
		}
	}
	// log.I.S(fmt.Sprint(decryptedStr), fmt.Sprint(ws.Pending.Load()))
	// if fmt.Sprint(decryptedStr) == fmt.Sprint(ws.Pending.Load()) {
	// 	return
	// }
	return
}

func (rl *Relay) command(ev *event.T, cmd string) (err error) {
	log.D.S(cmd)
	reply := MakeReply(ev, fmt.Sprintf("received message: %v", spew.Sdump(cmd)))
	log.I.F("executing command '%s' - currently only echo", cmd)
	if reply, err = EncryptDM(reply, rl.Config.SecKey, ev.PubKey); chk.E(err) {
		return
	}
	log.I.Ln("reply", reply.ToObject().String())
	rl.BroadcastEvent(reply)
	return
}
