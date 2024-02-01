package replicatr

import (
	"crypto/rand"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

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

func (rl *Relay) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	var err error
	var conn *websocket.Conn
	conn, err = rl.upgrader.Upgrade(w, r, nil)
	if rl.E.Chk(err) {
		rl.E.F("failed to upgrade websocket: %v", err)
		return
	}
	rl.clients.Store(conn, struct{}{})
	ticker := time.NewTicker(rl.PingPeriod)
	// NIP-42 challenge
	challenge := make([]byte, 8)
	_, err = rand.Read(challenge)
	rl.E.Chk(err)
	rem := r.Header.Get("X-Forwarded-For")
	splitted := strings.Split(rem, " ")
	var rr string
	if len(splitted) == 1 {
		rr = splitted[0]
	}
	if len(splitted) == 2 {
		rr = splitted[1]
	}
	// in case upstream doesn't set this or we are directly ristening instead of
	// via reverse proxy or just if the header field is missing, put the
	// connection remote address into the websocket state data.
	if rr == "" {
		rr = r.RemoteAddr
	}
	ws := &relayws.WebSocket{
		Conn:       conn,
		RealRemote: rr,
		Request:    r,
		Challenge:  hex.Enc(challenge),
		Authed:     make(chan struct{}),
	}
	c, cancel := context.Cancel(
		context.Value(
			context.Bg(),
			wsKey, ws,
		),
	)
	kill := func() {
		for _, onDisconnect := range rl.OnDisconnect {
			onDisconnect(c)
		}
		ticker.Stop()
		cancel()
		if _, ok := rl.clients.Load(conn); ok {
			_ = conn.Close()
			rl.clients.Delete(conn)
			RemoveListener(ws)
		}
	}
	go rl.websocketReadMessages(readParams{c, kill, ws, conn, r})
	go rl.websocketWatcher(c, kill, ticker, ws)
}

func (rl *Relay) wsProcessMessages(msg []byte, c context.T, ws *relayws.WebSocket) {
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
	if log.Fail(err) {
		return
	}
	if en == nil {
		rl.T.Ln("'silently' ignoring message")
		return
	}
	// rl.D.Ln("received envelope from", ws.conn.LocalAddr(), ws.conn.RemoteAddr())
	switch env := en.(type) {
	case *eventenvelope.T:
		// reject old dated events (eg running branle) todo: allow for authed
		if env.Event.CreatedAt <= 1640305962 {
			rl.T.F("rejecting event with date: %s", env.Event.CreatedAt.Time().String())
			rl.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "invalid: date is too far in the past"}))
			return
		}
		rl.T.Ln("event envelope")
		// check id
		evs := env.Event.ToCanonical().Bytes()
		rl.T.F("serialized %s", evs)
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
		rl.T.Ln("ID was valid")
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
			rl.E.Chk(ws.WriteEnvelope(&okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "invalid: signature is invalid"}))
			return
		}
		rl.T.Ln("signature was valid")
		if env.Event.Kind == kind.Deletion {
			// this always returns "blocked: " whenever it returns an error
			err = rl.handleDeleteRequest(c, env.Event)
		} else {
			rl.D.Ln("adding event")
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
		rl.D.Ln("sent back ok envelope")
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
			if rl.E.Chk(err) {
				// fail everything if any filter is rejected
				reason := err.Error()
				if strings.HasPrefix(reason, nip42.AuthRequired) {
					RequestAuth(c)
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
	log.Fail(err)
}

type readParams struct {
	c    context.T
	kill func()
	ws   *relayws.WebSocket
	conn *websocket.Conn
	r    *http.Request
}

func (rl *Relay) websocketReadMessages(p readParams) {
	defer p.kill()
	deny := true
	if len(rl.Whitelist) > 0 {
		for i := range rl.Whitelist {
			if rl.Whitelist[i] == p.ws.RealRemote {
				deny = false
			}
		}
	} else {
		deny = false
	}
	if deny {
		rl.T.F("denying access to '%s': dropping message", p.ws.RealRemote)
		return
	}
	p.conn.SetReadLimit(rl.MaxMessageSize)
	rl.E.Chk(p.conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
	p.conn.SetPongHandler(func(string) (err error) {
		err = p.conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		rl.E.Chk(err)
		return
	})
	for _, onConnect := range rl.OnConnect {
		onConnect(p.c)
	}
	for {
		var err error
		var typ int
		var message []byte
		typ, message, err = p.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,    // 1000
				websocket.CloseGoingAway,        // 1001
				websocket.CloseNoStatusReceived, // 1005
				websocket.CloseAbnormalClosure,  // 1006
			) {
				rl.E.F("unexpected close error from %s: %v",
					p.r.Header.Get("X-Forwarded-For"), err)
			}
			return
		}
		if typ == websocket.PingMessage {
			rl.E.Chk(p.ws.WriteMessage(websocket.PongMessage, nil))
			continue
		}
		trunc := make([]byte, 512)
		copy(trunc, message)
		var ellipsis string
		if len(message) > 512 {
			ellipsis = "..."
		}
		log.D.F("receiving message from '%s'\n%s%s",
			p.ws.RealRemote, string(trunc), ellipsis)
		go rl.wsProcessMessages(message, p.c, p.ws)
	}
}

func (rl *Relay) websocketWatcher(c context.T, kill func(), t *time.Ticker,
	ws *relayws.WebSocket) {

	var err error
	defer kill()
	for {
		select {
		case <-c.Done():
			return
		case <-t.C:
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
			if err = ws.WriteMessage(websocket.PingMessage, nil); rl.E.Chk(err) {
				if !strings.HasSuffix(err.Error(), "use of closed network connection") {
					rl.E.F("error writing ping: %v; closing websocket", err)
				}
				return
			}
		}
	}
}
