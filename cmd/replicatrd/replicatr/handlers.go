package replicatr

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip42"
	"github.com/rs/cors"
)

// ServeHTTP implements http.Handler interface.
func (rl *Relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rl.ServiceURL == "" {
		rl.ServiceURL = getServiceBaseURL(r)
	}
	if r.Header.Get("Upgrade") == "websocket" {
		rl.HandleWebsocket(w, r)
	} else if r.Header.Get("Accept") == "application/nostr+json" {
		cors.AllowAll().Handler(http.HandlerFunc(rl.HandleNIP11)).ServeHTTP(w, r)
	} else {
		rl.serveMux.ServeHTTP(w, r)
	}
}

func (rl *Relay) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	var e error
	var conn *websocket.Conn
	conn, e = rl.upgrader.Upgrade(w, r, nil)
	if rl.Log.E.Chk(e) {
		rl.Log.E.F("failed to upgrade websocket: %v\n", e)
		return
	}
	rl.clients.Store(conn, struct{}{})
	ticker := time.NewTicker(rl.PingPeriod)
	// NIP-42 challenge
	challenge := make([]byte, 8)
	_, e = rand.Read(challenge)
	rl.Log.E.Chk(e)
	ws := &WebSocket{
		conn:      conn,
		Request:   r,
		Challenge: hex.EncodeToString(challenge),
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
			rl.Log.E.Chk(conn.Close())
			rl.clients.Delete(conn)
			removeListener(ws)
		}
	}
	go rl.readMessages(ctx, kill, ws, conn, r)
	go rl.watcher(ctx, kill, ticker, ws)
}

func (rl *Relay) processMessages(message []byte, ctx context.Context, ws *WebSocket) {
	var e error
	envelope := nostr.ParseMessage(message)
	if envelope == nil {
		// stop silently
		return
	}
	switch env := envelope.(type) {
	case *nostr.EventEnvelope:
		// check id
		hash := sha256.Sum256(env.Event.Serialize())
		id := hex.EncodeToString(hash[:])
		if id != env.Event.ID {
			rl.Log.E.Chk(ws.WriteJSON(nostr.OKEnvelope{
				EventID: env.Event.ID,
				OK:      false,
				Reason:  "invalid: id is computed incorrectly",
			}))
			return
		}
		// check signature
		var ok bool
		if ok, e = env.Event.CheckSignature(); rl.Log.E.Chk(e) {
			rl.Log.E.Chk(ws.WriteJSON(nostr.OKEnvelope{
				EventID: env.Event.ID,
				OK:      false,
				Reason:  "error: failed to verify signature"},
			))
			return
		} else if !ok {
			rl.Log.E.Chk(ws.WriteJSON(nostr.OKEnvelope{
				EventID: env.Event.ID,
				OK:      false,
				Reason:  "invalid: signature is invalid"},
			))
			return
		}
		var writeErr error
		if env.Event.Kind == 5 {
			// this always returns "blocked: " whenever it returns an error
			writeErr = rl.handleDeleteRequest(ctx, &env.Event)
		} else {
			// this will also always return a prefixed reason
			writeErr = rl.AddEvent(ctx, &env.Event)
		}
		var reason string
		if !rl.Log.E.Chk(writeErr) {
			ok = true
		} else {
			reason = writeErr.Error()
			if strings.HasPrefix(reason, "auth-required:") {
				RequestAuth(ctx)
			}
		}
		rl.Log.E.Chk(ws.WriteJSON(nostr.OKEnvelope{
			EventID: env.Event.ID,
			OK:      ok,
			Reason:  reason,
		}))
	case *nostr.CountEnvelope:
		if rl.CountEvents == nil {
			rl.Log.E.Chk(ws.WriteJSON(nostr.ClosedEnvelope{
				SubscriptionID: env.SubscriptionID,
				Reason:         "unsupported: this relay does not support NIP-45"},
			))
			return
		}
		var total int64
		for _, filter := range env.Filters {
			total += rl.handleCountRequest(ctx, ws, &filter)
		}
		rl.Log.E.Chk(ws.WriteJSON(nostr.CountEnvelope{
			SubscriptionID: env.SubscriptionID,
			Count:          &total,
		}))
	case *nostr.ReqEnvelope:
		eose := sync.WaitGroup{}
		eose.Add(len(env.Filters))
		// a context just for the "stored events" request handler
		reqCtx, cancelReqCtx := context.WithCancelCause(ctx)
		// expose subscription id in the context
		reqCtx = context.WithValue(reqCtx, subscriptionIdKey, env.SubscriptionID)
		// handle each filter separately -- dispatching events as they're loaded from databases
		for _, filter := range env.Filters {
			e = rl.handleRequest(reqCtx, env.SubscriptionID, &eose, ws, &filter)
			if rl.Log.E.Chk(e) {
				// fail everything if any filter is rejected
				reason := e.Error()
				if strings.HasPrefix(reason, "auth-required:") {
					RequestAuth(ctx)
				}
				rl.Log.E.Chk(ws.WriteJSON(nostr.ClosedEnvelope{
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
			rl.Log.E.Chk(ws.WriteJSON(nostr.EOSEEnvelope(env.SubscriptionID)))
		}()
		setListener(env.SubscriptionID, ws, env.Filters, cancelReqCtx)
	case *nostr.CloseEnvelope:
		removeListenerId(ws, string(*env))
	case *nostr.AuthEnvelope:
		wsBaseUrl := strings.Replace(rl.ServiceURL, "http", "ws", 1)
		if pubkey, ok := nip42.ValidateAuthEvent(&env.Event, ws.Challenge, wsBaseUrl); ok {
			ws.AuthedPublicKey = pubkey
			ws.authLock.Lock()
			if ws.Authed != nil {
				close(ws.Authed)
				ws.Authed = nil
			}
			ws.authLock.Unlock()
			rl.Log.E.Chk(ws.WriteJSON(nostr.OKEnvelope{
				EventID: env.Event.ID,
				OK:      true},
			))
		} else {
			rl.Log.E.Chk(ws.WriteJSON(nostr.OKEnvelope{
				EventID: env.Event.ID,
				OK:      false,
				Reason:  "error: failed to authenticate"},
			))
		}
	}
}
func (rl *Relay) readMessages(ctx context.Context, kill func(), ws *WebSocket, conn *websocket.Conn, r *http.Request) {
	defer kill()
	conn.SetReadLimit(rl.MaxMessageSize)
	rl.Log.E.Chk(conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
	conn.SetPongHandler(func(string) (e error) {
		e = conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		rl.Log.E.Chk(e)
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
		if rl.Log.E.Chk(e) {
			if websocket.IsUnexpectedCloseError(
				e,
				websocket.CloseNormalClosure,    // 1000
				websocket.CloseGoingAway,        // 1001
				websocket.CloseNoStatusReceived, // 1005
				websocket.CloseAbnormalClosure,  // 1006
			) {
				rl.Log.E.F("unexpected close error from %s: %v\n",
					r.Header.Get("X-Forwarded-For"), e)
			}
			return
		}
		if typ == websocket.PingMessage {
			rl.Log.E.Chk(ws.WriteMessage(websocket.PongMessage, nil))
			continue
		}
		go rl.processMessages(message, ctx, ws)
	}
}

func (rl *Relay) watcher(ctx context.Context, kill func(), ticker *time.Ticker, ws *WebSocket) {
	var e error
	defer kill()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if e = ws.WriteMessage(websocket.PingMessage, nil); rl.Log.E.Chk(e) {
				if !strings.HasSuffix(e.Error(), "use of closed network connection") {
					rl.Log.E.F("error writing ping: %v; closing websocket\n", e)
				}
				return
			}
		}
	}
}

func (rl *Relay) HandleNIP11(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/nostr+json")
	info := *rl.Info
	for _, ovw := range rl.OverwriteRelayInformation {
		info = ovw(r.Context(), r, info)
	}
	rl.Log.E.Chk(json.NewEncoder(w).Encode(info))
}
