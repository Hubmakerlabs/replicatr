package khatru

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/nbd-wtf/go-nostr"
	"github.com/rs/cors"
	"mleku.dev/git/nostr/auth"
	"mleku.dev/git/nostr/envelopes"
	"mleku.dev/git/nostr/envelopes/authenvelope"
	"mleku.dev/git/nostr/envelopes/closeenvelope"
	"mleku.dev/git/nostr/envelopes/countenvelope"
	"mleku.dev/git/nostr/envelopes/eventenvelope"
	"mleku.dev/git/nostr/envelopes/okenvelope"
	"mleku.dev/git/nostr/envelopes/reqenvelope"
	"mleku.dev/git/nostr/eventid"
	"mleku.dev/git/nostr/interfaces/enveloper"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

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
	conn, err := rl.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.E.F("failed to upgrade websocket: %v\n", err)
		return
	}
	rl.clients.Store(conn, struct{}{})
	ticker := time.NewTicker(rl.PingPeriod)

	// NIP-42 challenge
	challenge := make([]byte, 8)
	rand.Read(challenge)

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
		for _, ondisconnect := range rl.OnDisconnect {
			ondisconnect(ctx)
		}

		ticker.Stop()
		cancel()
		if _, ok := rl.clients.Load(conn); ok {
			conn.Close()
			rl.clients.Delete(conn)
			removeListener(ws)
		}
	}

	go func() {
		defer kill()

		conn.SetReadLimit(rl.MaxMessageSize)
		conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(rl.PongWait))
			return nil
		})

		for _, onconnect := range rl.OnConnect {
			onconnect(ctx)
		}

		for {
			typ, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(
					err,
					websocket.CloseNormalClosure,    // 1000
					websocket.CloseGoingAway,        // 1001
					websocket.CloseNoStatusReceived, // 1005
					websocket.CloseAbnormalClosure,  // 1006
					4537,                            // some client seems to send many of these
				) {
					log.E.F("unexpected close error from %s: %v\n", r.Header.Get("X-Forwarded-For"), err)
				}
				return
			}

			if typ == websocket.PingMessage {
				ws.WriteMessage(websocket.PongMessage, nil)
				continue
			}

			go func(msg []byte) {
				var env enveloper.I
				if env, _, err = envelopes.ProcessEnvelope(msg); chk.I(err) {
					if env == nil {
						log.E.F("nil envelope label: ignoring message\n%s", string(msg))
						// ws.OffenseCount.Inc()
						// kill()
						// chk.E(ws.Conn.Close())
						return
					}
					// kill()
					return
				}

				switch env := env.(type) {
				case *eventenvelope.T:
					// check id
					hash := sha256.Sum256(env.Event.ToCanonical().Bytes())
					id := eventid.T(hex.EncodeToString(hash[:]))
					if id != env.Event.ID {
						ws.WriteMessage(websocket.TextMessage, (&okenvelope.T{ID: env.Event.ID, OK: false,
							Reason: "invalid: id is computed incorrectly"}).Bytes())
						return
					}

					// check signature
					if ok, err := env.Event.CheckSignature(); err != nil {
						ws.WriteMessage(websocket.TextMessage, (&okenvelope.T{ID: env.Event.ID, OK: false,
							Reason: "error: failed to verify signature"}).Bytes())
						return
					} else if !ok {
						ws.WriteMessage(websocket.TextMessage, (&okenvelope.T{ID: env.Event.ID, OK: false,
							Reason: "invalid: signature is invalid"}).Bytes())
						return
					}

					// check NIP-70 protected
					for _, v := range env.Event.Tags {
						if len(v) == 1 && v[0] == "-" {
							msg := "must be published by event author"
							authed := GetAuthed(ctx)
							if authed == "" {
								RequestAuth(ctx)
								ws.WriteMessage(websocket.TextMessage, (&okenvelope.T{
									ID:     env.Event.ID,
									OK:     false,
									Reason: "auth-required: " + msg,
								}).Bytes())
								return
							}
							if authed != env.Event.PubKey {
								ws.WriteMessage(websocket.TextMessage, (&okenvelope.T{
									ID:     env.Event.ID,
									OK:     false,
									Reason: "blocked: " + msg,
								}).Bytes())
								return
							}
						}
					}

					var ok bool
					var writeErr error
					if env.Event.Kind == 5 {
						// this always returns "blocked: " whenever it returns an error
						writeErr = rl.handleDeleteRequest(ctx, env.Event)
					} else {
						// this will also always return a prefixed reason
						writeErr = rl.AddEvent(ctx, env.Event)
					}

					var reason string
					if writeErr == nil {
						ok = true
						for _, ovw := range rl.OverwriteResponseEvent {
							ovw(ctx, env.Event)
						}
						notifyListeners(env.Event)
					} else {
						reason = writeErr.Error()
						if strings.HasPrefix(reason, "auth-required:") {
							RequestAuth(ctx)
						}
					}
					ws.WriteMessage(websocket.TextMessage,
						(&okenvelope.T{ID: env.Event.ID, OK: ok, Reason: reason}).Bytes())
				case *countenvelope.Request:
					if rl.CountEvents == nil {
						ws.WriteJSON(nostr.ClosedEnvelope{SubscriptionID: env.ID.String(),
							Reason: "unsupported: this relay does not support NIP-45"})
						return
					}
					var total int
					for _, filter := range env.Filters {
						total += rl.handleCountRequest(ctx, ws, *filter)
					}
					ws.WriteJSON(countenvelope.Response{ID: env.ID, Count: total})
				case *reqenvelope.T:
					eose := sync.WaitGroup{}
					eose.Add(len(env.Filters))

					// a context just for the "stored events" request handler
					reqCtx, cancelReqCtx := context.WithCancelCause(ctx)

					// expose subscription id in the context
					reqCtx = context.WithValue(reqCtx, subscriptionIdKey, env.SubscriptionID)

					// handle each filter separately -- dispatching events as they're loaded from databases
					for _, filter := range env.Filters {
						err := rl.handleRequest(reqCtx, env.SubscriptionID.String(), &eose, ws, *filter)
						if err != nil {
							// fail everything if any filter is rejected
							reason := err.Error()
							if strings.HasPrefix(reason, "auth-required:") {
								RequestAuth(ctx)
							}
							ws.WriteJSON(nostr.ClosedEnvelope{SubscriptionID: env.SubscriptionID.String(),
								Reason: reason})
							cancelReqCtx(errors.New("filter rejected"))
							return
						}
					}

					go func() {
						// when all events have been loaded from databases and dispatched
						// we can cancel the context and fire the EOSE message
						eose.Wait()
						cancelReqCtx(nil)
						ws.WriteJSON(nostr.EOSEEnvelope(env.SubscriptionID))
					}()

					setListener(env.SubscriptionID.String(), ws, env.Filters, cancelReqCtx)
				case *closeenvelope.T:
					removeListenerId(ws, env.String())
				case *authenvelope.Response:
					wsBaseUrl := strings.Replace(rl.ServiceURL, "http", "ws", 1)
					var ok bool
					var pubkey string
					if pubkey, ok, err = auth.Validate(env.Event, ws.Challenge, wsBaseUrl); ok {
						ws.AuthedPublicKey = pubkey
						ws.authLock.Lock()
						if ws.Authed != nil {
							close(ws.Authed)
							ws.Authed = nil
						}
						ws.authLock.Unlock()
						ws.WriteMessage(websocket.TextMessage,
							(&okenvelope.T{ID: env.Event.ID, OK: true}).Bytes())
					} else {
						ws.WriteMessage(websocket.TextMessage,
							(&okenvelope.T{ID: env.Event.ID, OK: false,
								Reason: "error: failed to authenticate"}).Bytes())
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
				err := ws.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					if !strings.HasSuffix(err.Error(), "use of closed network connection") {
						log.E.F("error writing ping: %v; closing websocket", err)
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

	json.NewEncoder(w).Encode(info)
}
