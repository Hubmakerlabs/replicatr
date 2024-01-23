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
	ws := &WebSocket{
		conn:      conn,
		Request:   r,
		Challenge: hex.Enc(challenge),
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
	go rl.websocketReadMessages(c, kill, ws, conn, r)
	go rl.websocketWatcher(c, kill, ticker, ws)
}

func (rl *Relay) wsProcessMessages(msg []byte, c context.T, ws *WebSocket) {
	rl.T.F("processing message '%s", string(msg))
	en, _, err := envelopes.ProcessEnvelope(msg)
	if log.Fail(err) {
		return
	}
	if en == nil {
		rl.T.Ln("'silently' ignoring message")
		return
	}
	switch env := en.(type) {
	case *eventenvelope.T:
		rl.T.Ln("event envelope")
		// check id
		evs := env.Event.ToCanonical().Bytes()
		rl.T.F("serialized %s", evs)
		hash := sha256.Sum256(evs)
		id := hex.Enc(hash[:])
		if id != env.Event.ID.String() {
			rl.T.F("id mismatch got %s, expected %s",
				id, env.Event.ID.String())
			rl.E.Chk(ws.WriteJSON(okenvelope.T{
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
			rl.E.Chk(ws.WriteJSON(okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "error: failed to verify signature: " + err.Error()},
			))
			return
		} else if !ok {
			rl.E.Chk(ws.WriteJSON(okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "invalid: signature is invalid"},
			))
			return
		}
		rl.T.Ln("signature was valid")
		var writeErr error
		if env.Event.Kind == kind.Deletion {
			// this always returns "blocked: " whenever it returns an error
			writeErr = rl.handleDeleteRequest(c, env.Event)
		} else {
			rl.D.Ln("adding event")
			// this will also always return a prefixed reason
			writeErr = rl.AddEvent(c, env.Event)
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
		rl.D.Ln("sending back ok envelope")
		rl.E.Chk(ws.WriteJSON(okenvelope.T{
			ID:     env.Event.ID,
			OK:     ok,
			Reason: reason,
		}))
		rl.D.Ln("sent back ok envelope")
	case *countenvelope.Request:
		if rl.CountEvents == nil {
			rl.E.Chk(ws.WriteJSON(closedenvelope.T{
				ID:     env.ID,
				Reason: "unsupported: this relay does not support NIP-45",
			}))
			return
		}
		var total int64
		for _, f := range env.Filters {
			total += rl.handleCountRequest(c, ws, f)
		}
		rl.E.Chk(ws.WriteJSON(countenvelope.Response{
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
		// handle each filter separately -- dispatching events as they're loaded from databases
		for _, f := range env.Filters {
			err = rl.handleFilter(reqCtx, env.SubscriptionID.String(), &wg, ws, f)
			if rl.E.Chk(err) {
				// fail everything if any filter is rejected
				reason := err.Error()
				if strings.HasPrefix(reason, "auth-required:") {
					RequestAuth(c)
				}
				rl.E.Chk(ws.WriteJSON(closedenvelope.T{
					ID:     env.SubscriptionID,
					Reason: reason},
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
			rl.E.Chk(ws.WriteJSON(eoseenvelope.T{T: env.SubscriptionID}))
		}()
		SetListener(env.SubscriptionID.String(), ws, env.Filters, cancelReqCtx)
	case *closeenvelope.T:
		RemoveListenerId(ws, env.T.String())
	case *authenvelope.Response:
		wsBaseUrl := strings.Replace(rl.ServiceURL, "http", "ws", 1)
		if pubkey, ok := nip42.ValidateAuthEvent(env.Event, ws.Challenge, wsBaseUrl); ok {
			ws.AuthedPublicKey = pubkey
			ws.authLock.Lock()
			if ws.Authed != nil {
				close(ws.Authed)
				ws.Authed = nil
			}
			ws.authLock.Unlock()
			rl.E.Chk(ws.WriteJSON(okenvelope.T{
				ID: env.Event.ID,
				OK: true,
			}))
		} else {
			rl.E.Chk(ws.WriteJSON(okenvelope.T{
				ID:     env.Event.ID,
				OK:     false,
				Reason: "error: failed to authenticate"},
			))
		}
	}
}

func (rl *Relay) websocketReadMessages(c context.T, kill func(),
	ws *WebSocket, conn *websocket.Conn, r *http.Request) {

	defer kill()
	conn.SetReadLimit(rl.MaxMessageSize)
	rl.E.Chk(conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
	conn.SetPongHandler(func(string) (err error) {
		err = conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		rl.E.Chk(err)
		return
	})
	for _, onConnect := range rl.OnConnect {
		onConnect(c)
	}
	for {
		var err error
		var typ int
		var message []byte
		typ, message, err = conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,    // 1000
				websocket.CloseGoingAway,        // 1001
				websocket.CloseNoStatusReceived, // 1005
				websocket.CloseAbnormalClosure,  // 1006
			) {
				rl.E.F("unexpected close error from %s: %v",
					r.Header.Get("X-Forwarded-For"), err)
			}
			return
		}
		if typ == websocket.PingMessage {
			rl.E.Chk(ws.WriteMessage(websocket.PongMessage, nil))
			continue
		}
		go rl.wsProcessMessages(message, c, ws)
	}
}

func (rl *Relay) websocketWatcher(c context.T, kill func(), t *time.Ticker, ws *WebSocket) {
	var err error
	defer kill()
	for {
		select {
		case <-c.Done():
			return
		case <-t.C:
			if err = ws.WriteMessage(websocket.PingMessage, nil); rl.E.Chk(err) {
				if !strings.HasSuffix(err.Error(), "use of closed network connection") {
					rl.E.F("error writing ping: %v; closing websocket", err)
				}
				return
			}
		}
	}
}
