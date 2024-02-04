package replicatr

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
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip42"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/fasthttp/websocket"
	"github.com/minio/sha256-simd"
)

func (rl *Relay) wsProcessMessages(msg []byte, c context.T,
	kill func(), ws *relayws.WebSocket) {

	if len(msg) > rl.Info.Limitation.MaxMessageLength {
		rl.T.F("rejecting event with size: %d", len(msg))
		rl.E.Chk(ws.WriteEnvelope(&okenvelope.T{
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
			if rl.Whitelist[i] == ws.RealRemote {
				deny = false
			}
		}
	} else {
		deny = false
	}
	if deny {
		rl.T.F("denying access to '%s': dropping message", ws.RealRemote)
		return
	}
	en, _, err := envelopes.ProcessEnvelope(msg)
	if rl.T.Chk(err) {
		return
	}
	if en == nil {
		rl.T.Ln("'silently' ignoring message")
		return
	}
	// rl.D.Ln("received envelope from", ws.conn.LocalAddr(), ws.conn.RemoteAddr())
	switch env := en.(type) {
	case *eventenvelope.T:
		// reject old dated events according to nip11
		if env.Event.CreatedAt <= rl.Info.Limitation.Oldest {
			rl.T.F("rejecting event with date: %s", env.Event.CreatedAt.Time().String())
			rl.E.Chk(ws.WriteEnvelope(&okenvelope.T{
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
		// rl.T.F("serialized %s", evs)
		hash := sha256.Sum256(evs)
		id := hex.Enc(hash[:])
		if id != env.Event.ID.String() {
			rl.T.F("id mismatch got %s, expected %s",
				id, env.Event.ID.String())
			rl.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "invalid: id is computed incorrectly",
			}))
			return
		}
		// check signature
		var ok bool
		if ok, err = env.Event.CheckSignature(); rl.E.Chk(err) {
			rl.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "error: failed to verify signature: " + err.Error(),
			}))
			return
		} else if !ok {
			rl.E.Ln("invalid: signature is invalid")
			rl.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "invalid: signature is invalid"}))
			return
		}
		if env.Event.Kind == kind.Deletion {
			// this always returns "blocked: " whenever it returns an error
			err = rl.handleDeleteRequest(c, env.Event)
		} else {
			rl.T.Ln("adding event")
			// this will also always return a prefixed reason
			err = rl.AddEvent(c, env.Event)
		}
		var reason string
		if ok = !rl.E.Chk(err); !ok {
			reason = err.Error()
			if strings.HasPrefix(reason, nip42.AuthRequired) {
				RequestAuth(c)
			}
		} else {
			ok = true
		}
		rl.T.Ln("sending back ok envelope")
		rl.E.Chk(ws.WriteEnvelope(&okenvelope.T{
			ID:     env.Event.ID,
			OK:     ok,
			Reason: reason,
		}))
	case *countenvelope.Request:
		if rl.CountEvents == nil {
			rl.E.Chk(ws.WriteEnvelope(&closedenvelope.T{
				ID:     env.ID,
				Reason: "unsupported: this relay does not support NIP-45",
			}))
			return
		}
		var total int64
		for _, f := range env.Filters {
			total += rl.handleCountRequest(c, ws, f)
		}
		rl.E.Chk(ws.WriteEnvelope(&countenvelope.Response{
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
			// if we are not given a limit we will be stingy and only return 5
			// results
			if f.Limit == 0 {
				f.Limit = 5
			}
			err = rl.handleFilter(handleFilterParams{
				reqCtx,
				env.SubscriptionID,
				&wg,
				ws,
				f,
			})
			if rl.T.Chk(err) {
				// fail everything if any filter is rejected
				reason := err.Error()
				if strings.HasPrefix(reason, nip42.AuthRequired) {
					RequestAuth(c)
				}
				if strings.HasPrefix(reason, "blocked") {
					kill()
					return
				}
				rl.E.Chk(ws.WriteEnvelope(&closedenvelope.T{
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
			rl.E.Chk(ws.WriteEnvelope(&eoseenvelope.T{T: env.SubscriptionID}))
		}()
		SetListener(env.SubscriptionID.String(), ws, env.Filters, cancelReqCtx)
	case *closeenvelope.T:
		RemoveListenerId(ws, env.T.String())
	case *authenvelope.Response:
		wsBaseUrl := strings.Replace(rl.ServiceURL, "http", "ws", 1)
		var ok bool
		var pubkey string
		if pubkey, ok, err = nip42.ValidateAuthEvent(env.Event, ws.Challenge, wsBaseUrl); ok {
			ws.AuthPubKey = pubkey
			ws.Authed <- struct{}{}
			rl.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID: env.Event.ID,
				OK: true,
			}))
		} else {
			rl.E.Chk(ws.WriteMessage(
				websocket.TextMessage, (&okenvelope.T{
					ID:     env.Event.ID,
					OK:     false,
					Reason: "error: failed to authenticate"}).
					Bytes(),
			))
		}
	}
	rl.Fail(err)
}
