package app

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/authenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closedenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eoseenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/minio/sha256-simd"
)

func (rl *Relay) processEventEnvelope(msg []byte, env *eventenvelope.T,
	c context.T, ws *relayws.WebSocket, serviceURL string) (err error) {

	var ok bool
	if !rl.IsAuthed(c, "EVENT") {
		return
	}
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
		log.D.F("id mismatch got %s, expected %s %s %s\n%s\n%s",
			ws.RealRemote(),
			ws.AuthPubKey(), id, env.Event.ID.String(),
			env.Event.ToObject().String(),
			string(msg))
		chk.E(ws.WriteEnvelope(&okenvelope.T{
			ID:     env.Event.ID,
			OK:     false,
			Reason: "invalid: id is computed incorrectly",
		}))
		return
	}
	// check signature
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
		// this will also always return a prefixed reason
		err = rl.AddEvent(c, env.Event)
	}
	var reason string
	if err != nil {
		reason = err.Error()
		if strings.HasPrefix(reason, auth.Required) {
			log.I.Ln("requesting auth")
			RequestAuth(c, env.Label())
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

	return
}

func (rl *Relay) processCountEnvelope(msg []byte, env *countenvelope.Request,
	c context.T, ws *relayws.WebSocket, serviceURL string) (err error) {

	if !rl.IsAuthed(c, "COUNT") {
		return
	}
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
	return
}

func (rl *Relay) processReqEnvelope(msg []byte, env *reqenvelope.T,
	c context.T, ws *relayws.WebSocket, serviceURL string) (err error) {

	if !rl.IsAuthed(c, "REQ") {
		return
	}
	wg := sync.WaitGroup{}
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
				RequestAuth(c, env.Label())
			}
			if strings.HasPrefix(reason, "blocked") {
				return
			}
			chk.E(ws.WriteEnvelope(&closedenvelope.T{
				ID:     env.SubscriptionID,
				Reason: reason,
			}))
			log.I.Ln("cancelling req context")
			cancelReqCtx(errors.New("filter rejected"))
			return
		}
	}
	go func() {
		// when all events have been loaded from databases and dispatched
		// we can cancel the context and fire the EOSE message
		wg.Wait()
		// log.I.Ln("cancelling req context")
		cancelReqCtx(nil)
		chk.E(ws.WriteEnvelope(&eoseenvelope.T{Sub: env.SubscriptionID}))
	}()
	SetListener(env.SubscriptionID.String(), ws, env.Filters, cancelReqCtx)
	return
}

func (rl *Relay) processAuthEnvelope(msg []byte, env *authenvelope.Response,
	c context.T, ws *relayws.WebSocket, serviceURL string) (err error) {

	log.T.Ln("received auth response")
	wsBaseUrl := strings.Replace(serviceURL, "http", "ws", 1)
	var ok bool
	var pubkey string
	if pubkey, ok, err = auth.Validate(env.Event, ws.Challenge(),
		wsBaseUrl); ok {
		if ws.AuthPubKey() == env.Event.PubKey {
			log.D.Ln("user already authed")
			return
		}
		log.I.Ln("user authenticated", pubkey)
		ws.SetAuthPubKey(pubkey)
		log.I.Ln("closing auth chan")
		close(ws.Authed)
		chk.E(ws.WriteEnvelope(&okenvelope.T{
			ID: env.Event.ID,
			OK: true,
		}))
		return
	} else {
		log.E.Ln("user sent bogus auth response")
		chk.E(ws.WriteEnvelope(&okenvelope.T{
			ID:     env.Event.ID,
			OK:     false,
			Reason: "error: failed to authenticate"}),
		)
	}
	return
}
