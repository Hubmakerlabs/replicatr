package relay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip42"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip45"
	"github.com/fasthttp/websocket"
	"github.com/minio/sha256-simd"
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
	if conn, e = rl.upgrader.Upgrade(w, r, nil); fails(e) {
		rl.Log.E.F("failed to upgrade websocket: %v\n", e)
		return
	}
	rl.clients.Store(conn, struct{}{})
	ticker := time.NewTicker(rl.PingPeriod)

	// NIP-42 challenge
	challenge := make([]byte, 8)

	var n int
	if n, e = rand.Read(challenge); fails(e) {
		rl.Log.E.F("only read %d bytes from system CSPRNG", n)
	}
	ws := &WebSocket{
		conn:      conn,
		Request:   r,
		Challenge: hex.EncodeToString(challenge),
		Authed:    make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(
		context.WithValue(
			context.Background(),
			WebsocketContextKey, ws,
		),
	)
	kill := func() {
		for _, onDisconnect := range rl.OnDisconnect {
			onDisconnect(ctx)
		}
		ticker.Stop()
		cancel()
		if _, ok := rl.clients.Load(conn); ok {
			rl.Log.D.Chk(conn.Close())
			rl.clients.Delete(conn)
			removeListener(ws)
		}
	}
	go func() {
		defer kill()
		conn.SetReadLimit(rl.MaxMessageSize)
		rl.Log.E.Chk(conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
		conn.SetPongHandler(func(string) (e error) {
			rl.Log.E.Chk(conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
			return nil
		})
		for _, onConnect := range rl.OnConnect {
			onConnect(ctx)
		}
		var e error
		for {
			var typ int
			var message []byte
			typ, message, e = conn.ReadMessage()
			if fails(e) {
				if websocket.IsUnexpectedCloseError(
					e,
					websocket.CloseNormalClosure,    // 1000
					websocket.CloseGoingAway,        // 1001
					websocket.CloseNoStatusReceived, // 1005
					websocket.CloseAbnormalClosure,  // 1006
				) {
					rl.Log.E.F("unexpected close error from %s: %v\n", r.Header.Get("X-Forwarded-For"), e)
				}
				return
			}
			rl.Log.D.F("received message on websocket: '%s'", string(message))
			if typ == websocket.PingMessage {
				rl.Log.D.Chk(ws.WriteMessage(websocket.PongMessage, nil))
				continue
			}
			go func(message []byte) {
				var e error
				var ok bool
				var envelope nip1.Enveloper
				if envelope, _, _, e = nip1.ProcessEnvelope(message); fails(e) || envelope == nil {
					return
				}
				switch env := envelope.(type) {
				case *nip1.EventEnvelope:
					// check id
					hash := sha256.Sum256(env.Event.ToCanonical().Bytes())
					id := hex.EncodeToString(hash[:])
					if nip1.EventID(id) != env.Event.ID {
						rl.Log.E.Chk(ws.WriteJSON(nip1.OKEnvelope{
							EventID: env.Event.ID,
							OK:      false,
							Reason:  "invalid: id is computed incorrectly",
						}))
						return
					}
					// check signature
					if ok, e = env.Event.CheckSignature(); rl.Log.E.Chk(e) {
						rl.Log.E.Chk(ws.WriteJSON(nip1.OKEnvelope{
							EventID: env.Event.ID,
							OK:      false,
							Reason:  "error: failed to verify signature",
						}))
						return
					} else if !ok {
						rl.Log.E.Chk(ws.WriteJSON(nip1.OKEnvelope{
							EventID: env.Event.ID,
							OK:      false,
							Reason:  "invalid: signature is invalid",
						}))
						return
					}
					var ok bool
					var writeErr error
					if env.Event.Kind == 5 {
						// this always returns "blocked: " whenever it returns
						// an error
						writeErr = rl.handleDeleteRequest(ctx, env.Event)
					} else {
						// this will also always return a prefixed reason
						writeErr = rl.AddEvent(ctx, env.Event)
					}
					var reason string
					if writeErr == nil {
						ok = true
					} else {
						reason = writeErr.Error()
						if strings.HasPrefix(reason, "auth-required:") {
							RequestAuth(ctx)
						}
					}
					rl.Log.D.Chk(ws.WriteJSON(nip1.OKEnvelope{
						EventID: env.Event.ID, OK: ok, Reason: reason}))
				case *nip45.CountRequestEnvelope:
					if rl.CountEvents == nil {
						rl.Log.E.Chk(ws.WriteJSON(nip1.ClosedEnvelope{
							SubscriptionID: env.SubscriptionID,
							Reason: "unsupported: " +
								"this relay does not support NIP-45",
						}))
						return
					}
					var total int64
					for _, filter := range env.Filters {
						total += rl.handleCountRequest(ctx, ws, filter)
					}
					rl.Log.D.Chk(ws.WriteJSON(nip45.CountResponseEnvelope{
						SubscriptionID: env.SubscriptionID,
						Count:          total,
					}))
				case *nip1.ReqEnvelope:
					eose := sync.WaitGroup{}
					eose.Add(len(env.Filters))
					// a context just for the "stored events" request handler
					reqCtx, cancelReqCtx := context.WithCancelCause(ctx)
					// expose subscription id in the context
					reqCtx = context.WithValue(reqCtx, SubscriptionIDContextKey,
						env.SubscriptionID)
					// handle each filter separately -- dispatching events as
					// they're loaded from databases
					for _, filter := range env.Filters {
						e = rl.handleRequest(reqCtx, env.SubscriptionID,
							&eose, ws, filter)
						if e != nil {
							// fail everything if any filter is rejected
							reason := e.Error()
							if strings.HasPrefix(reason, "auth-required:") {
								RequestAuth(ctx)
							}
							rl.Log.D.Chk(ws.WriteJSON(nip1.ClosedEnvelope{
								SubscriptionID: env.SubscriptionID,
								Reason:         reason,
							}))
							cancelReqCtx(errors.New("filter rejected"))
							return
						}
					}
					go func() {
						// when all events have been loaded from databases and
						// dispatched we can cancel the context and fire the
						// EOSE message
						eose.Wait()
						cancelReqCtx(nil)
						rl.Log.E.Chk(ws.WriteJSON(nip1.EOSEEnvelope{
							SubscriptionID: env.SubscriptionID}))
					}()

					setListener(env.SubscriptionID, ws, env.Filters,
						cancelReqCtx)
				case *nip1.CloseEnvelope:
					removeListenerId(ws, env.SubscriptionID)
				case *nip42.AuthResponseEnvelope:
					wsBaseUrl := strings.Replace(rl.ServiceURL, "http",
						"ws", 1)
					if pubkey, ok := nip42.ValidateAuthEvent(env.Event,
						ws.Challenge, wsBaseUrl); ok {

						ws.AuthedPublicKey = pubkey
						close(ws.Authed)
						rl.Log.E.Chk(ws.WriteJSON(nip1.OKEnvelope{
							EventID: env.Event.ID, OK: true}))
					} else {
						rl.Log.E.Chk(ws.WriteJSON(nip1.OKEnvelope{
							EventID: env.Event.ID,
							OK:      false,
							Reason:  "error: failed to authenticate",
						}))
					}
				}
			}(message)
		}
	}()

	go func() {
		defer kill()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e := ws.WriteMessage(websocket.PingMessage, nil)
				if e != nil {
					if !strings.HasSuffix(e.Error(),
						"use of closed network connection",
					) {
						rl.Log.E.F("error writing ping: %v; "+
							"closing websocket\n", e)
					}
					return
				}
			}
		}
	}()
}

func (rl *Relay) HandleNIP11(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/nostr+json")
	info := rl.Info
	for _, ovw := range rl.OverwriteRelayInformation {
		info = ovw(r.Context(), r, info)
	}
	rl.Log.E.Chk(json.NewEncoder(w).Encode(info))
}
