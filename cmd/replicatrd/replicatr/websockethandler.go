package replicatr

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip42"
	"github.com/fasthttp/websocket"
)

func (rl *Relay) HandleWebsocket(w ResponseWriter, r *Request) {
	var e error
	var conn *Conn
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
	ctx, cancel := context.WithCancel(
		context.WithValue(
			context.Background(),
			wsKey, ws,
		),
	)
	kill := func() {
		for _, onDisconnect := range rl.OnDisconnect {
			onDisconnect(ctx)
		}
		ticker.Stop()
		cancel()
		if _, ok := rl.clients.Load(conn); ok {
			_ = conn.Close()
			rl.clients.Delete(conn)
			removeListener(ws)
		}
	}
	go rl.websocketReadMessages(ctx, kill, ws, conn, r)
	go rl.websocketWatcher(ctx, kill, ticker, ws)
}

func (rl *Relay) websocketProcessMessages(message []byte, ctx Ctx, ws *WebSocket) {
	var e error
	envelope := nostr.ParseMessage(message)
	if envelope == nil {
		// stop silently
		return
	}
	switch env := envelope.(type) {
	case *nostr.EventEnvelope:
		// check id
		hash := sha256.Sum256(env.T.Serialize())
		id := hex.EncodeToString(hash[:])
		if id != env.T.ID {
			rl.E.Chk(ws.WriteJSON(OKEnvelope{
				EventID: env.T.ID,
				OK:      false,
				Reason:  "invalid: id is computed incorrectly",
			}))
			return
		}
		// check signature
		var ok bool
		if ok, e = env.T.CheckSignature(); rl.E.Chk(e) {
			rl.E.Chk(ws.WriteJSON(OKEnvelope{
				EventID: env.T.ID,
				OK:      false,
				Reason:  "error: failed to verify signature"},
			))
			return
		} else if !ok {
			rl.E.Chk(ws.WriteJSON(OKEnvelope{
				EventID: env.T.ID,
				OK:      false,
				Reason:  "invalid: signature is invalid"},
			))
			return
		}
		var writeErr error
		if env.T.Kind == 5 {
			// this always returns "blocked: " whenever it returns an error
			writeErr = rl.handleDeleteRequest(ctx, &env.T)
		} else {
			// this will also always return a prefixed reason
			writeErr = rl.AddEvent(ctx, &env.T)
		}
		var reason string
		if ok = !rl.E.Chk(writeErr); !ok {
			reason = writeErr.Error()
			if strings.HasPrefix(reason, "auth-required:") {
				RequestAuth(ctx)
			}
		} else {
			ok = true
		}
		rl.E.Chk(ws.WriteJSON(OKEnvelope{
			EventID: env.T.ID,
			OK:      ok,
			Reason:  reason,
		}))
	case *CountEnvelope:
		if rl.CountEvents == nil {
			rl.E.Chk(ws.WriteJSON(ClosedEnvelope{
				SubscriptionID: env.SubscriptionID,
				Reason:         "unsupported: this relay does not support NIP-45",
			}))
			return
		}
		var total int64
		for _, filter := range env.Filters {
			total += rl.handleCountRequest(ctx, ws, &filter)
		}
		rl.E.Chk(ws.WriteJSON(CountEnvelope{
			SubscriptionID: env.SubscriptionID,
			Count:          &total,
		}))
	case *ReqEnvelope:
		eose := WaitGroup{}
		eose.Add(len(env.Filters))
		// a context just for the "stored events" request handler
		reqCtx, cancelReqCtx := context.WithCancelCause(ctx)
		// expose subscription id in the context
		reqCtx = context.WithValue(reqCtx, subscriptionIdKey, env.SubscriptionID)
		// handle each filter separately -- dispatching events as they're loaded from databases
		for _, filter := range env.Filters {
			e = rl.handleFilter(reqCtx, env.SubscriptionID, &eose, ws, &filter)
			if rl.E.Chk(e) {
				// fail everything if any filter is rejected
				reason := e.Error()
				if strings.HasPrefix(reason, "auth-required:") {
					RequestAuth(ctx)
				}
				rl.E.Chk(ws.WriteJSON(ClosedEnvelope{
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
			eose.Wait()
			cancelReqCtx(nil)
			rl.E.Chk(ws.WriteJSON(EOSEEnvelope(env.SubscriptionID)))
		}()
		setListener(env.SubscriptionID, ws, env.Filters, cancelReqCtx)
	case *CloseEnvelope:
		removeListenerId(ws, string(*env))
	case *AuthEnvelope:
		wsBaseUrl := strings.Replace(rl.ServiceURL, "http", "ws", 1)
		if pubkey, ok := nip42.ValidateAuthEvent(&env.Event, ws.Challenge, wsBaseUrl); ok {
			ws.AuthedPublicKey = pubkey
			ws.authLock.Lock()
			if ws.Authed != nil {
				close(ws.Authed)
				ws.Authed = nil
			}
			ws.authLock.Unlock()
			rl.E.Chk(ws.WriteJSON(OKEnvelope{
				EventID: env.Event.ID,
				OK:      true},
			))
		} else {
			rl.E.Chk(ws.WriteJSON(OKEnvelope{
				EventID: env.Event.ID,
				OK:      false,
				Reason:  "error: failed to authenticate"},
			))
		}
	}
}

func (rl *Relay) websocketReadMessages(ctx Ctx, kill func(),
	ws *WebSocket, conn *Conn, r *Request) {

	defer kill()
	conn.SetReadLimit(rl.MaxMessageSize)
	rl.E.Chk(conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
	conn.SetPongHandler(func(string) (e error) {
		e = conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		rl.E.Chk(e)
		return
	})
	for _, onConnect := range rl.OnConnect {
		onConnect(ctx)
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
		go rl.websocketProcessMessages(message, ctx, ws)
	}
}

func (rl *Relay) websocketWatcher(ctx Ctx, kill func(), ticker *time.Ticker, ws *WebSocket) {
	var e error
	defer kill()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if e = ws.WriteMessage(websocket.PingMessage, nil); rl.E.Chk(e) {
				if !strings.HasSuffix(e.Error(), "use of closed network connection") {
					rl.E.F("error writing ping: %v; closing websocket\n", e)
				}
				return
			}
		}
	}
}
