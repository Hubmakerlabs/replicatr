package app

import (
	"fmt"
	"strconv"
	"strings"

	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/crypt"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/tags"
	"mleku.dev/git/nostr/timestamp"
)

// DecryptDM decrypts a DM, kind 4, 1059 or 1060
func DecryptDM(ev *event.T, meSec, youPub string) (decryptedStr string, err error) {
	switch ev.Kind {
	case kind.EncryptedDirectMessage:
		var secret, decrypted []byte
		if secret, err = crypt.ComputeSharedSecret(meSec, youPub); chk.E(err) {
			return
		}
		if decrypted, err = crypt.Decrypt(ev.Content, secret); chk.E(err) {
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
		if secret, err = crypt.ComputeSharedSecret(meSec, youPub); chk.E(err) {
			return
		}
		if ev.Content, err = crypt.Encrypt(ev.Content, secret); chk.E(err) {
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
	ws := GetConnection(c)
	if ws == nil {
		return
	}
	log.D.Ln("running chat checker")
	if ev.Kind != kind.EncryptedDirectMessage {
		log.T.Ln("not chat event", ev.Kind, kind.GetString(ev.Kind))
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
	log.T.F("decrypted message: '%s'", decryptedStr)
	decryptedStr = strings.TrimSpace(decryptedStr)
	var reply *event.T
	if ws.AuthPubKey() == "" {
		if strings.HasPrefix(decryptedStr, "AUTH_") {
			var authed bool
			authStr := strings.Split(decryptedStr, "_")
			log.I.Ln(authStr, ws.Challenge())
			if len(authStr) == 3 {
				var ts int64
				if ts, err = strconv.ParseInt(authStr[1], 10, 64); chk.E(err) {
					return
				}
				now := timestamp.Now().Time().Unix()
				log.I.Ln()
				if ts < now+15 || ts > now-15 {
					if authStr[2] == ws.Challenge() {
						authed = true
						ws.SetAuthPubKey(ev.PubKey)
					}
				}
			}
			if !authed {
				reply = MakeReply(ev, fmt.Sprintf("access denied"))
				if reply, err = EncryptDM(reply, meSec, youPub); chk.E(err) {
					return
				}
				log.T.Ln("reply", reply.ToObject().String())
				rl.BroadcastEvent(reply)
				ws.GenerateChallenge()
				return
			} else {
				// now process cached
				log.T.Ln("pending message:", ws.Pending.Load())
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
`, decryptedStr, timestamp.Now(), ws.Challenge())
			log.I.F("sending message to user\n%s", content)
			reply = MakeReply(ev, content)
			if reply, err = EncryptDM(reply, meSec, youPub); chk.E(err) {
				return
			}
			log.T.Ln("reply", reply.ToObject().String())
			rl.BroadcastEvent(reply)
			return
		}
	} else {
		if err = rl.command(ev, decryptedStr); chk.E(err) {
			return
		}
	}
	return
}

type Command struct {
	Name string
	Help string
	Func func(rl *Relay, prefix string, ev *event.T, cmd *Command, args ...string) (reply *event.T, err error)
}

func (rl *Relay) command(ev *event.T, cmd string) (err error) {
	args := strings.Split(cmd, " ")
	if len(args) < 1 {
		err = log.E.Err("no command received")
		return
	}
	var reply *event.T
	for i := range Commands {
		if Commands[i].Name == args[0] {
			if reply, err = Commands[i].Func(rl, "", ev, Commands[i], args...); chk.E(err) {
				return
			}
			break
		}
	}
	if reply == nil {
		for i := range Commands {
			if Commands[i].Name == "help" {
				reply, err = help(rl, fmt.Sprintf("unknown command: '%s'", cmd),
					ev, Commands[i], args...)
				if chk.E(err) {
					return
				}
				break
			}
		}
	}
	if reply, err = EncryptDM(reply, rl.Config.SecKey, ev.PubKey); chk.E(err) {
		return
	}
	rl.BroadcastEvent(reply)
	return
}
