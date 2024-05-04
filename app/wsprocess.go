package app

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/authenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closedenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eoseenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/fasthttp/websocket"
	"github.com/minio/sha256-simd"
)

const IgnoreAfter = 16

func (rl *Relay) wsProcessMessages(msg []byte, c context.T,
	kill func(), ws *relayws.WebSocket) (err error) {

	if len(msg) == 0 {
		err = log.E.Err("empty message, probably dropped connection")
		return
	}
	if ws.OffenseCount.Load() > IgnoreAfter {
		err = log.E.Err("client keeps sending wrong req envelopes")
		return
	}
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
			Reason: fmt.Sprintf(
				"invalid: relay limit disallows messages larger than %d bytes,"+
					" this message is %d bytes",
				rl.Info.Limitation.MaxMessageLength, len(msg)),
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
		if en == nil {
			log.E.F("nil envelope label: ignoring message\n%s", string(msg))
			ws.OffenseCount.Inc()
			return
		}
		return
	}
	switch env := en.(type) {
	case *eventenvelope.T:
		// reject old dated events according to nip11
		if env.Event.CreatedAt <= rl.Info.Limitation.Oldest {
			log.D.F("rejecting event with date: %s %s %s",
				env.Event.CreatedAt.Time().String(), ws.RealRemote(),
				ws.AuthPubKey())
			chk.E(ws.WriteEnvelope(&okenvelope.T{
				ID: env.Event.ID,
				OK: false,
				Reason: fmt.Sprintf(
					"invalid: relay limit disallows timestamps older than %d",
					rl.Info.Limitation.Oldest),
			}))
			return
		}
		// check id
		evs := env.Event.ToCanonical().Bytes()
		hash := sha256.Sum256(evs)
		id := hex.Enc(hash[:])
		if id != env.Event.ID.String() {
			log.D.F("id mismatch got %s, expected %s %s %s", ws.RealRemote(),
				ws.AuthPubKey(), id, env.Event.ID.String())
			chk.E(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "invalid: id is computed incorrectly",
			}))
			return
		}
		// check signature
		var ok bool
		if ok, err = env.Event.CheckSignature(); chk.E(err) {
			chk.E(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "error: failed to verify signature: " + err.Error(),
			}))
			return
		} else if !ok {
			log.E.Ln("invalid: signature is invalid", ws.RealRemote(),
				ws.AuthPubKey())
			chk.E(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "invalid: signature is invalid"}))
			return
		}
		if env.Event.Kind == kind.Deletion {
			// this always returns "blocked: " whenever it returns an error
			err = rl.handleDeleteRequest(c, env.Event)
		} else {
			// log.D.Ln("adding event", ws.AuthPubKey(),
			// 	ws.RealRemote(), env.Event.ToObject().String())
			// this will also always return a prefixed reason
			err = rl.AddEvent(c, env.Event)
		}
		var reason string
		if ok = !chk.E(err); !ok {
			reason = err.Error()
			if strings.HasPrefix(reason, auth.Required) {
				log.I.Ln("requesting auth")
				RequestAuth(c)
				ok = true
			}
			if strings.HasPrefix(reason, "duplicate") {
				ok = true
			}
		} else {
			ok = true
		}
		chk.E(ws.WriteEnvelope(&okenvelope.T{
			ID:     env.Event.ID,
			OK:     ok,
			Reason: reason,
		}))
	case *countenvelope.Request:
		if rl.CountEvents == nil {
			chk.E(ws.WriteEnvelope(&closedenvelope.T{
				ID:     env.ID,
				Reason: "unsupported: this relay does not support NIP-45",
			}))
			return
		}
		var total int
		for _, f := range env.Filters {
			total += rl.handleCountRequest(c, env.ID, ws, f)
		}
		chk.E(ws.WriteEnvelope(&countenvelope.Response{
			ID:    env.ID,
			Count: total,
		}))
	case *reqenvelope.T:
		wg := sync.WaitGroup{}
		wg.Add(len(env.Filters))
		// a context just for the "stored events" request handler
		reqCtx, cancelReqCtx := context.CancelCause(c)
		// expose subscription id in the context
		reqCtx = context.Value(reqCtx, subscriptionIdKey, env.SubscriptionID)
		// handle each filter separately -- dispatching events as they're loaded
		// from databases
		for _, f := range env.Filters {
			err = rl.handleFilter(handleFilterParams{
				reqCtx,
				env.SubscriptionID,
				&wg,
				ws,
				f,
			})
			if log.T.Chk(err) {
				// fail everything if any filter is rejected
				reason := err.Error()
				if strings.HasPrefix(reason, auth.Required) {
					RequestAuth(c)
				}
				if strings.HasPrefix(reason, "blocked") {
					return
				}
				chk.E(ws.WriteEnvelope(&closedenvelope.T{
					ID:     env.SubscriptionID,
					Reason: reason,
				}))
				cancelReqCtx(errors.New("filter rejected"))
				return
			}
		}
		go func() {
			// when all events have been loaded from databases and dispatched
			// we can cancel the context and fire the EOSE message
			wg.Wait()
			cancelReqCtx(nil)
			chk.E(ws.WriteEnvelope(&eoseenvelope.T{Sub: env.SubscriptionID}))
		}()
		SetListener(env.SubscriptionID.String(), ws, env.Filters, cancelReqCtx)
	case *closeenvelope.T:
		RemoveListenerId(ws, env.T.String())
	case *authenvelope.Response:
		log.T.Ln("received auth response")
		wsBaseUrl := strings.Replace(rl.ServiceURL.Load(), "http", "ws", 1)
		var ok bool
		var pubkey string
		if pubkey, ok, err = auth.Validate(env.Event, ws.Challenge(),
			wsBaseUrl); ok {
			if ws.AuthPubKey() == env.Event.PubKey {
				log.D.Ln("user already authed")
				break
			}
			log.I.Ln("user authenticated", pubkey)
			ws.SetAuthPubKey(pubkey)
			close(ws.Authed)
			chk.E(ws.WriteEnvelope(&okenvelope.T{
				ID: env.Event.ID,
				OK: true,
			}))
			return
		} else {
			log.E.Ln("user sent bogus auth response")
			chk.E(ws.WriteMessage(
				websocket.TextMessage, (&okenvelope.T{
					ID:     env.Event.ID,
					OK:     false,
					Reason: "error: failed to authenticate"}).
					Bytes(),
			))
		}
	}
	return
}
