package app

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/authenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
)

const IgnoreAfter = 16

func (rl *Relay) wsProcessMessages(msg []byte, c context.T,
	kill func(), ws *relayws.WebSocket, serviceURL string) (err error) {

	if len(msg) == 0 {
		err = log.E.Err("empty message, probably dropped connection")
		return
	}
	if ws.OffenseCount.Load() > IgnoreAfter {
		err = log.E.Err("client keeps sending wrong req envelopes")
		return
	}
	// log.I.F("websocket receive message\n%s\n%s %s",
	//  string(msg), ws.RealRemote(), ws.AuthPubKey())
	strMsg := string(msg)
	if ws.OffenseCount.Load() > IgnoreAfter {
		if len(strMsg) > 256 {
			strMsg = strMsg[:256]
		}
		log.T.Ln("dropping message due to over", IgnoreAfter,
			"errors from this client on this connection",
			ws.RealRemote(), ws.AuthPubKey(), strMsg)
		return
	}
	if len(msg) > rl.Info.Limitation.MaxMessageLength {
		log.D.F("rejecting event with size: %d from %s %s",
			len(msg), ws.RealRemote(), ws.AuthPubKey())
		chk.E(ws.WriteEnvelope(&okenvelope.T{
			OK: false,
			Reason: normalize.Reason(okenvelope.Invalid.S(),
				fmt.Sprintf(
					"relay limit disallows messages larger than %d "+
						"bytes, this message is %d bytes",
					rl.Info.Limitation.MaxMessageLength, len(msg))),
		}))
		return
	}
	deny := true
	if len(rl.Whitelist) > 0 {
		for i := range rl.Whitelist {
			if rl.Whitelist[i] == ws.RealRemote() {
				deny = false
			}
		}
	} else {
		deny = false
	}
	if deny {
		log.E.F("denying access to '%s' %s: dropping message", ws.RealRemote(),
			ws.AuthPubKey())
		return
	}
	var en enveloper.I
	if en, _, err = envelopes.ProcessEnvelope(msg); log.E.Chk(err) {
		chk.E(ws.WriteEnvelope(&okenvelope.T{
			OK: false,
			Reason: normalize.Reason(fmt.Sprintf(
				"malformed JSON, possibly invalid unicode escapes\n%s",
				string(msg)),
				okenvelope.Invalid.S()),
		}))
		return fmt.Errorf("invalid: error processing envelope: %s", err.Error())
	}
	switch env := en.(type) {
	case *eventenvelope.T:
		if err = rl.processEventEnvelope(msg, env, c, ws,
			serviceURL); chk.E(err) {
			return
		}
	case *countenvelope.Request:
		if err = rl.processCountEnvelope(msg, env, c, ws,
			serviceURL); chk.E(err) {
			return
		}
	case *reqenvelope.T:
		if err = rl.processReqEnvelope(msg, env, c, ws,
			serviceURL); chk.E(err) {
			return
		}
	case *closeenvelope.T:
		RemoveListenerId(ws, env.T.String())
	case *authenvelope.Response:
		if err = rl.processAuthEnvelope(msg, env, c, ws,
			serviceURL); chk.E(err) {
			return
		}
	}
	return
}
