package replicatr

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/OK"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/closed"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/closer"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/count"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelope"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip42"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/req"
	"github.com/fasthttp/websocket"
)

func (rl *Relay) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	var e error
	var conn *websocket.Conn
	conn, e = rl.upgrader.Upgrade(w, r, nil)
	if rl.E.Chk(e) {
		rl.E.F("failed to upgrade websocket: %v\n", e)
		return
	}
	rl.clients.Store(conn, struct{}{})
	ticker := time.NewTicker(rl.PingPeriod)
	// NIP-42 challenge
	challenge := make([]byte, 8)
	_, e = rand.Read(challenge)
	rl.E.Chk(e)
	ws := &WebSocket{
		conn:      conn,
		Request:   r,
		Challenge: encodeToHex(challenge),
	}
	c, cancel := context.WithCancel(
		context.WithValue(
			context.Background(),
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
	go rl.websocketReadMessages(c, kill, ws, conn, r)
	go rl.websocketWatcher(c, kill, ticker, ws)
}

func (rl *Relay) websocketProcessMessages(message []byte, c context.Context, ws *WebSocket) {
	var e error
	env := envelope.ParseMessage(message)
	if env == nil {
		// stop silently
		return
	}
	switch env := env.(type) {
	case *event.Envelope:
		// check id
		hash := sha256.Sum256(env.T.Serialize())
		id := hex.EncodeToString(hash[:])
		if id != env.T.ID {
			rl.E.Chk(ws.WriteJSON(OK.Envelope{
				EventID: env.T.ID,
				OK:      false,
				Reason:  "invalid: id is computed incorrectly",
			}))
			return
		}
		// check signature
		var ok bool
		if ok, e = env.T.CheckSignature(); rl.E.Chk(e) {
			rl.E.Chk(ws.WriteJSON(OK.Envelope{
				EventID: env.T.ID,
				OK:      false,
				Reason:  "error: failed to verify signature"},
			))
			return
		} else if !ok {
			rl.E.Chk(ws.WriteJSON(OK.Envelope{
				EventID: env.T.ID,
				OK:      false,
				Reason:  "invalid: signature is invalid"},
			))
			return
		}
		var writeErr error
		if env.T.Kind == 5 {
			// this always returns "blocked: " whenever it returns an error
			writeErr = rl.handleDeleteRequest(c, &env.T)
		} else {
			// this will also always return a prefixed reason
			writeErr = rl.AddEvent(c, &env.T)
		}
		var reason string
		if ok = !rl.E.Chk(writeErr); !ok {
			reason = writeErr.Error()
			if strings.HasPrefix(reason, "auth-required:") {
				RequestAuth(c)
			}
		} else {
			ok = true
		}
		rl.E.Chk(ws.WriteJSON(OK.Envelope{
			EventID: env.T.ID,
			OK:      ok,
			Reason:  reason,
		}))
	case *count.Envelope:
		if rl.CountEvents == nil {
			rl.E.Chk(ws.WriteJSON(closed.Envelope{
				SubscriptionID: env.SubscriptionID,
				Reason:         "unsupported: this relay does not support NIP-45",
			}))
			return
		}
		var total int64
		for _, f := range env.T {
			total += rl.handleCountRequest(c, ws, &f)
		}
		rl.E.Chk(ws.WriteJSON(count.Envelope{
			SubscriptionID: env.SubscriptionID,
			Count:          &total,
		}))
	case *req.Envelope:
		wg := sync.WaitGroup{}
		wg.Add(len(env.T))
		// a context just for the "stored events" request handler
		reqCtx, cancelReqCtx := context.WithCancelCause(c)
		// expose subscription id in the context
		reqCtx = context.WithValue(reqCtx, subscriptionIdKey, env.SubscriptionID)
		// handle each filter separately -- dispatching events as they're loaded from databases
		for _, f := range env.T {
			e = rl.handleFilter(reqCtx, env.SubscriptionID, &wg, ws, &f)
			if rl.E.Chk(e) {
				// fail everything if any filter is rejected
				reason := e.Error()
				if strings.HasPrefix(reason, "auth-required:") {
					RequestAuth(c)
				}
				rl.E.Chk(ws.WriteJSON(closed.Envelope{
					SubscriptionID: env.SubscriptionID,
					Reason:         reason},
				))
				cancelReqCtx(errors.New("filter rejected"))
				return
			}
		}
		go func() {
			// when all events have been loaded from databases and dispatched
			// we can cancel the context and fire the EOSE message
			wg.Wait()
			cancelReqCtx(nil)
			rl.E.Chk(ws.WriteJSON(eose.Envelope(env.SubscriptionID)))
		}()
		SetListener(env.SubscriptionID, ws, env.T, cancelReqCtx)
	case *closer.Envelope:
		RemoveListenerId(ws, string(*env))
	case *auth.Envelope:
		wsBaseUrl := strings.Replace(rl.ServiceURL, "http", "ws", 1)
		if pubkey, ok := nip42.ValidateAuthEvent(&env.Event, ws.Challenge, wsBaseUrl); ok {
			ws.AuthedPublicKey = pubkey
			ws.authLock.Lock()
			if ws.Authed != nil {
				close(ws.Authed)
				ws.Authed = nil
			}
			ws.authLock.Unlock()
			rl.E.Chk(ws.WriteJSON(OK.Envelope{
				EventID: env.Event.ID,
				OK:      true},
			))
		} else {
			rl.E.Chk(ws.WriteJSON(OK.Envelope{
				EventID: env.Event.ID,
				OK:      false,
				Reason:  "error: failed to authenticate"},
			))
		}
	}
}

func (rl *Relay) websocketReadMessages(c context.Context, kill func(),
	ws *WebSocket, conn *websocket.Conn, r *http.Request) {

	defer kill()
	conn.SetReadLimit(rl.MaxMessageSize)
	rl.E.Chk(conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
	conn.SetPongHandler(func(string) (e error) {
		e = conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		rl.E.Chk(e)
		return
	})
	for _, onConnect := range rl.OnConnect {
		onConnect(c)
	}
	for {
		var e error
		var typ int
		var message []byte
		typ, message, e = conn.ReadMessage()
		if e != nil {
			if websocket.IsUnexpectedCloseError(
				e,
				websocket.CloseNormalClosure,    // 1000
				websocket.CloseGoingAway,        // 1001
				websocket.CloseNoStatusReceived, // 1005
				websocket.CloseAbnormalClosure,  // 1006
			) {
				rl.E.F("unexpected close error from %s: %v\n",
					r.Header.Get("X-Forwarded-For"), e)
			}
			return
		}
		if typ == websocket.PingMessage {
			rl.E.Chk(ws.WriteMessage(websocket.PongMessage, nil))
			continue
		}
		go rl.websocketProcessMessages(message, c, ws)
	}
}

func (rl *Relay) websocketWatcher(c context.Context, kill func(), t *time.Ticker, ws *WebSocket) {
	var e error
	defer kill()
	for {
		select {
		case <-c.Done():
			return
		case <-t.C:
			if e = ws.WriteMessage(websocket.PingMessage, nil); rl.E.Chk(e) {
				if !strings.HasSuffix(e.Error(), "use of closed network connection") {
					rl.E.F("error writing ping: %v; closing websocket\n", e)
				}
				return
			}
		}
	}
}
