package app

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/authenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closedenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eoseenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip42"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/fasthttp/websocket"
	"github.com/minio/sha256-simd"
)

func (rl *Relay) wsProcessMessages(msg []byte, c context.T,
	kill func(), ws *relayws.WebSocket) {

	log.T.Ln("processing message", ws.RealRemote.Load(), string(msg))
	if len(msg) > rl.Info.Limitation.MaxMessageLength {
		log.D.F("rejecting event with size: %d", len(msg))
		log.E.Chk(ws.WriteEnvelope(&okenvelope.T{
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
			if rl.Whitelist[i] == ws.RealRemote.Load() {
				deny = false
			}
		}
	} else {
		deny = false
	}
	if deny {
		log.E.F("denying access to '%s': dropping message", ws.RealRemote.Load())
		return
	}
	var en enveloper.I
	var err error
	if en, _, err = envelopes.ProcessEnvelope(msg); log.D.Chk(err) {
		return
	}
	if en == nil {
		log.E.Ln("'silently' ignoring message")
		return
	}
	switch env := en.(type) {
	case *eventenvelope.T:
		log.D.Ln("received event envelope from",
			ws.RealRemote.Load(), en.ToArray().String())
		// reject old dated events according to nip11
		if env.Event.CreatedAt <= rl.Info.Limitation.Oldest {
			log.D.F("rejecting event with date: %s",
				env.Event.CreatedAt.Time().String())
			log.E.Chk(ws.WriteEnvelope(&okenvelope.T{
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
		// log.D.F("serialized %s", evs)
		hash := sha256.Sum256(evs)
		id := hex.Enc(hash[:])
		if id != env.Event.ID.String() {
			log.D.F("id mismatch got %s, expected %s",
				id, env.Event.ID.String())
			log.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "invalid: id is computed incorrectly",
			}))
			return
		}
		// check signature
		var ok bool
		if ok, err = env.Event.CheckSignature(); log.E.Chk(err) {
			log.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "error: failed to verify signature: " + err.Error(),
			}))
			return
		} else if !ok {
			log.E.Ln("invalid: signature is invalid")
			log.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "invalid: signature is invalid"}))
			return
		}
		if env.Event.Kind == kind.Deletion {
			// this always returns "blocked: " whenever it returns an error
			err = rl.handleDeleteRequest(c, env.Event)
		} else {
			log.D.Ln("adding event", env.Event.ToObject().String())
			// this will also always return a prefixed reason
			err = rl.AddEvent(c, env.Event)
		}
		var reason string
		if ok = !log.E.Chk(err); !ok {
			reason = err.Error()
			if strings.HasPrefix(reason, nip42.AuthRequired) {
				RequestAuth(c)
			}
			if strings.HasPrefix(reason, "duplicate") {
				ok = true
			}
		} else {
			ok = true
		}
		log.D.Ln("sending back ok envelope")
		log.E.Chk(ws.WriteEnvelope(&okenvelope.T{
			ID:     env.Event.ID,
			OK:     ok,
			Reason: reason,
		}))
	case *countenvelope.Request:
		if rl.CountEvents == nil {
			log.E.Chk(ws.WriteEnvelope(&closedenvelope.T{
				ID:     env.ID,
				Reason: "unsupported: this relay does not support NIP-45",
			}))
			return
		}
		var total int64
		for _, f := range env.Filters {
			total += rl.handleCountRequest(c, ws, f)
		}
		log.E.Chk(ws.WriteEnvelope(&countenvelope.Response{
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
			// // if we are not given a limit we will be stingy and only return 5
			// // results
			// if f.Limit != nil && *f.Limit == 0 {
			// 	*f.Limit = 5
			// }
			err = rl.handleFilter(handleFilterParams{
				reqCtx,
				env.SubscriptionID,
				&wg,
				ws,
				f,
			})
			if log.D.Chk(err) {
				// fail everything if any filter is rejected
				reason := err.Error()
				if strings.HasPrefix(reason, nip42.AuthRequired) {
					RequestAuth(c)
				}
				if strings.HasPrefix(reason, "blocked") {
					// kill()
					return
				}
				log.E.Chk(ws.WriteEnvelope(&closedenvelope.T{
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
			log.E.Chk(ws.WriteEnvelope(&eoseenvelope.T{Sub: env.SubscriptionID}))
		}()
		SetListener(env.SubscriptionID.String(), ws, env.Filters, cancelReqCtx)
	case *closeenvelope.T:
		log.D.Ln("received close envelope from",
			ws.RealRemote.Load(), en.ToArray().String())
		RemoveListenerId(ws, env.T.String())
	case *authenvelope.Response:
		log.D.Ln("received auth response envelope from",
			ws.RealRemote.Load(), en.ToArray().String())
		// log.D.Ln("received auth response")
		wsBaseUrl := strings.Replace(rl.ServiceURL.Load(), "http", "ws", 1)
		var ok bool
		var pubkey string
		if pubkey, ok, err = nip42.ValidateAuthEvent(env.Event, ws.Challenge.Load(), wsBaseUrl); ok {
			ws.AuthPubKey.Store(pubkey)
			ws.Authed <- struct{}{}
			log.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID: env.Event.ID,
				OK: true,
			}))
		} else {
			log.E.Chk(ws.WriteMessage(
				websocket.TextMessage, (&okenvelope.T{
					ID:     env.Event.ID,
					OK:     false,
					Reason: "error: failed to authenticate"}).
					Bytes(),
			))
		}
	}
	log.D.Chk(err)
}
