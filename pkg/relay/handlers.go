package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/enveloper"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/OK"
	auth2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closed"
	close2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/req"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
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
	if conn, e = rl.upgrader.Upgrade(w, r, nil); log.Fail(e) {
		rl.E.F("failed to upgrade websocket: %v", e)
		return
	}
	rl.clients.Store(conn, struct{}{})
	ticker := time.NewTicker(rl.PingPeriod)

	// NIP-42 challenge
	challenge := make([]byte, 8)

	var n int
	if n, e = rand.Read(challenge); log.Fail(e) {
		rl.E.F("only read %d bytes from system CSPRNG", n)
	}
	ws := &WebSocket{
		conn:      conn,
		Request:   r,
		Challenge: hex.EncodeToString(challenge),
		Authed:    make(chan struct{}),
	}
	ctx, cancel := context.Cancel(
		context.Value(
			context.Bg(),
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
			rl.D.Chk(conn.Close())
			rl.clients.Delete(conn)
			removeListener(ws)
		}
	}
	go func() {
		defer kill()
		conn.SetReadLimit(rl.MaxMessageSize)
		rl.E.Chk(conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
		conn.SetPongHandler(func(string) (e error) {
			rl.E.Chk(conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
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
			if log.Fail(e) {
				if websocket.IsUnexpectedCloseError(
					e,
					websocket.CloseNormalClosure,    // 1000
					websocket.CloseGoingAway,        // 1001
					websocket.CloseNoStatusReceived, // 1005
					websocket.CloseAbnormalClosure,  // 1006
				) {
					rl.E.F("unexpected close error from %s: %v", r.Header.Get("X-Forwarded-For"), e)
				}
				return
			}
			rl.D.F("received message on websocket: '%s'", string(message))
			if typ == websocket.PingMessage {
				rl.D.Chk(ws.WriteMessage(websocket.PongMessage, nil))
				continue
			}
			go func(message []byte) {
				var e error
				var ok bool
				var envelope enveloper.I
				if envelope, _, e = envelopes.ProcessEnvelope(message); log.Fail(e) || envelope == nil {
					return
				}
				switch env := envelope.(type) {
				case *event.Envelope:
					// check id
					hash := sha256.Sum256(env.Event.ToCanonical().Bytes())
					id := hex.EncodeToString(hash[:])
					if eventid.EventID(id) != env.Event.ID {
						rl.E.Chk(ws.WriteJSON(OK.Envelope{
							EventID: env.Event.ID,
							OK:      false,
							Reason:  "invalid: id is computed incorrectly",
						}))
						return
					}
					// check signature
					if ok, e = env.Event.CheckSignature(); rl.E.Chk(e) {
						rl.E.Chk(ws.WriteJSON(OK.Envelope{
							EventID: env.Event.ID,
							OK:      false,
							Reason:  "error: failed to verify signature",
						}))
						return
					} else if !ok {
						rl.E.Chk(ws.WriteJSON(OK.Envelope{
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
					rl.D.Chk(ws.WriteJSON(OK.Envelope{
						EventID: env.Event.ID, OK: ok, Reason: reason}))
				case *countenvelope.Request:
					if rl.CountEvents == nil {
						rl.E.Chk(ws.WriteJSON(closed.Envelope{
							T: env.SubscriptionID,
							Reason: "unsupported: " +
								"this relay does not support NIP-45",
						}))
						return
					}
					var total int64
					for _, f := range env.T {
						total += rl.handleCountRequest(ctx, ws, f)
					}
					rl.D.Chk(ws.WriteJSON(countenvelope.Response{
						SubscriptionID: env.SubscriptionID,
						Count:          total,
					}))
				case *req.Envelope:
					ee := sync.WaitGroup{}
					ee.Add(len(env.T))
					// a context just for the "stored events" request handler
					reqCtx, cancelReqCtx := context.CancelCause(ctx)
					// expose subscription id in the context
					reqCtx = context.Value(reqCtx, SubscriptionIDContextKey,
						env.SubscriptionID)
					// handle each filter separately -- dispatching events as
					// they're loaded from databases
					for _, f := range env.T {
						e = rl.handleRequest(reqCtx, env.SubscriptionID,
							&ee, ws, f)
						if e != nil {
							// fail everything if any filter is rejected
							reason := e.Error()
							if strings.HasPrefix(reason, "auth-required:") {
								RequestAuth(ctx)
							}
							rl.D.Chk(ws.WriteJSON(closed.Envelope{
								T:      env.SubscriptionID,
								Reason: reason,
							}))
							cancelReqCtx(errors.New("filter rejected"))
							return
						}
					}
					go func() {
						// when all events have been loaded from databases and
						// dispatched we can cancel the context and fire the
						// EOSE message
						ee.Wait()
						cancelReqCtx(nil)
						rl.E.Chk(ws.WriteJSON(eose.Envelope{
							T: env.SubscriptionID}))
					}()

					setListener(env.SubscriptionID, ws, env.T,
						cancelReqCtx)
				case *close2.Envelope:
					removeListenerId(ws, env.T)
				case *auth2.Response:
					wsBaseUrl := strings.Replace(rl.ServiceURL, "http",
						"ws", 1)
					if pubkey, o := auth2.Validate(env.T,
						ws.Challenge, wsBaseUrl); o {

						ws.AuthedPublicKey = pubkey
						close(ws.Authed)
						rl.E.Chk(ws.WriteJSON(OK.Envelope{
							EventID: env.T.ID, OK: true}))
					} else {
						rl.E.Chk(ws.WriteJSON(OK.Envelope{
							EventID: env.T.ID,
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
						rl.E.F("error writing ping: %v; "+
							"closing websocket", e)
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
	rl.E.Chk(json.NewEncoder(w).Encode(info))
}
