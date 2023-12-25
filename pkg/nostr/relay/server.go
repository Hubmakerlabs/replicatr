package relay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip11"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip42"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip45"
	"github.com/fasthttp/websocket"
	"github.com/puzpuzpuz/xsync/v2"
	"github.com/rs/cors"
	"github.com/sebest/xff"
	log2 "mleku.online/git/log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type (
	RejectEvent func(
		ctx context.Context, event *nip1.Event) (reject bool, msg string)
	RejectFilter func(
		ctx context.Context, filter nip1.Filter) (reject bool, msg string)
	OverwriteDeletionOutcome func(
		ctx context.Context, target *nip1.Event,
		deletion *nip1.Event) (acceptDeletion bool, msg string)
	OverwriteResponseEvent    func(ctx context.Context, event *nip1.Event)
	OverwriteFilter           func(ctx context.Context, filter *nip1.Filter)
	OverwriteRelayInformation func(ctx context.Context, r *http.Request,
		info nip11.RelayInfo) nip11.RelayInfo
	StoreEvent  func(ctx context.Context, event *nip1.Event) (e error)
	QueryEvents func(
		ctx context.Context, filter nip1.Filter) (evc chan *nip1.Event, e error)
	CountEvents func(
		ctx context.Context, filter nip1.Filter) (count int64, e error)
	Hook       func(ctx context.Context)
	PubkeyHook func(ctx context.Context, pubkey string)
	EventHook  func(ctx context.Context, event *nip1.Event)
)

type Listener struct {
	filters nip1.Filters
	cancel  context.CancelCauseFunc
}

type Server struct {
	ServiceURL                string
	Info                      *nip11.RelayInfo
	Log                       *log2.Logger
	upgrader                  websocket.Upgrader
	clients                   *xsync.MapOf[*websocket.Conn, struct{}]
	Addr                      string
	serveMux                  *http.ServeMux
	httpServer                *http.Server
	listeners                 *xsync.MapOf[*WebSocket, *xsync.MapOf[string, *Listener]]
	WriteWait                 time.Duration // Time allowed to write a message
	PongWait                  time.Duration // Time allowed to read a pong
	PingPeriod                time.Duration // Time between pings (< PongWait)
	MaxMessageSize            int64         // Maximum message size allowed
	RejectEvent               []RejectEvent
	RejectFilter              []RejectFilter
	RejectCountFilter         []RejectFilter
	OverwriteDeletionOutcome  []OverwriteDeletionOutcome
	OverwriteResponseEvent    []OverwriteResponseEvent
	OverwriteFilter           []OverwriteFilter
	OverwriteCountFilter      []OverwriteFilter
	OverwriteRelayInformation []OverwriteRelayInformation
	StoreEvent                []StoreEvent
	DeleteEvent               []StoreEvent
	QueryEvents               []QueryEvents
	CountEvents               []CountEvents
	OnAuth                    []PubkeyHook
	OnConnect                 []Hook
	OnDisconnect              []Hook
	OnEventSaved              []EventHook
}

type WebSocket struct {
	conn  *websocket.Conn
	mutex sync.Mutex

	// original request
	Request *http.Request

	// nip42
	Challenge       string
	AuthedPublicKey string
	Authed          chan struct{}
}

func (ws *WebSocket) Write(b []byte) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.conn.WriteMessage(websocket.TextMessage, b)
}

func NewRelay() *Server {
	return &Server{
		Log: log2.GetLogger(),

		Info: &nip11.RelayInfo{
			Software:      "https://github.com/fiatjaf/khatru",
			Version:       "n/a",
			SupportedNIPs: make([]int, 0),
		},

		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},

		clients:  xsync.NewTypedMapOf[*websocket.Conn, struct{}](pointerHasher[websocket.Conn]),
		serveMux: &http.ServeMux{},

		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     30 * time.Second,
		MaxMessageSize: 512000,
	}
}

func (rl *Server) GetListeningFilters() (filters nip1.Filters) {
	filters = make(nip1.Filters, 0, rl.listeners.Size()*2)

	rl.listeners.Range(func(_ *WebSocket,
		subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(_ string, listener *Listener) bool {
			for _, listenerFilter := range listener.filters {
				for _, respFilter := range filters {
					// check if this filter specifically is already added to respfilters
					if nip1.FilterEqual(&listenerFilter, &respFilter) {
						goto nextConn
					}
				}

				// field not yet present on filters, add it
				filters = append(filters, listenerFilter)

				// continue to the next filter
			nextConn:
				continue
			}

			return true
		})

		return true
	})

	return
}

func (rl *Server) setListener(id nip1.SubscriptionID, ws *WebSocket,
	filters nip1.Filters, cancel context.CancelCauseFunc) {

	subs, _ := rl.listeners.LoadOrCompute(ws,
		func() *xsync.MapOf[string, *Listener] {
			return xsync.NewMapOf[*Listener]()
		})
	subs.Store(string(id), &Listener{filters: filters, cancel: cancel})
}

// remove a specific subscription id from listeners for a given ws client
// and cancel its specific context
func (rl *Server) removeListenerId(ws *WebSocket, id nip1.SubscriptionID) {
	if subs, ok := rl.listeners.Load(ws); ok {
		if listener, ok := subs.LoadAndDelete(string(id)); ok {
			listener.cancel(fmt.Errorf("subscription closed by client"))
		}
		if subs.Size() == 0 {
			rl.listeners.Delete(ws)
		}
	}
}

// remove WebSocket conn from listeners (no need to cancel contexts as they are
// all inherited from the main connection context)
func (rl *Server) removeListener(ws *WebSocket) { rl.listeners.Delete(ws) }

func (rl *Server) notifyListeners(event *nip1.Event) {

	rl.listeners.Range(func(ws *WebSocket,
		subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.filters.Match(event) {
				return true
			}
			ee, e := nip1.NewEventEnvelope(id, event)
			if rl.Log.E.Chk(e) {
				return false
			}
			if e = ws.Write(ee.Bytes()); rl.Log.E.Chk(e) {
				return false
			}
			return true
		})
		return true
	})
}

func GetConnection(ctx context.Context) *WebSocket {
	return ctx.Value(WS_KEY).(*WebSocket)
}

func GetAuthed(ctx context.Context) string {
	return GetConnection(ctx).AuthedPublicKey
}

func GetIP(ctx context.Context) string {
	return xff.GetRemoteAddr(GetConnection(ctx).Request)
}

// GetOpenSubscriptions returns the list of current filters being run on new
// events.
func (rl *Server) GetOpenSubscriptions(ctx context.Context) []nip1.Filter {

	if subs, ok := rl.listeners.Load(GetConnection(ctx)); ok {
		res := make([]nip1.Filter, 0, rl.listeners.Size()*2)
		subs.Range(func(_ string, sub *Listener) bool {
			res = append(res, sub.filters...)
			return true
		})
		return res
	}
	return nil
}

// AddEvent adds an event to
func (rl *Server) AddEvent(ctx context.Context, evt *nip1.Event) (e error) {

	if evt == nil {
		return errors.New("error: event is nil")
	}
	for _, rejectFn := range rl.RejectEvent {
		if reject, msg := rejectFn(ctx, evt); reject {
			if msg == "" {
				return errors.New("blocked: no reason")
			} else {
				return errors.New(nip1.OKMessage(nip1.OKBlocked, msg))
			}
		}
	}
	var ch chan *nip1.Event
	store := true
	switch k := evt.Kind; {
	case k.IsEphemeral():
		// do not store ephemeral events
		store = false
	case k.IsReplaceable():
		// replaceable event, delete before storing
		for _, query := range rl.QueryEvents {
			ch, e = query(ctx,
				nip1.Filter{Authors: []string{evt.PubKey},
					Kinds: kind.Array{evt.Kind}})
			if e != nil {
				continue
			}
			if previous := <-ch; previous != nil && IsOlder(previous, evt) {
				for _, del := range rl.DeleteEvent {
					e = del(ctx, previous)
					if e != nil {
						return e
					}
				}
			}
		}
	case k.IsParameterizedReplaceable():
		// parameterized replaceable event, delete before storing
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d != nil {
			for _, query := range rl.QueryEvents {
				ch, e = query(ctx,
					nip1.Filter{Authors: []string{evt.PubKey},
						Kinds: kind.Array{evt.Kind},
						Tags:  nip1.TagMap{"d": []string{d.Value()}}})
				if e != nil {
					continue
				}
				if previous := <-ch; previous != nil &&
					IsOlder(previous, evt) {

					for _, del := range rl.DeleteEvent {
						rl.Log.D.Chk(del(ctx, previous))
					}
				}
			}
		}
	}
	if store {
		for _, storeFn := range rl.StoreEvent {
			if e = storeFn(ctx, evt); e != nil {
				switch {
				case errors.Is(e, eventstore.ErrDupEvent):
					return nil
				default:
					return fmt.Errorf(nip1.OKMessage(nip1.OKError, e.Error()))
				}
			}
		}
		for _, ons := range rl.OnEventSaved {
			ons(ctx, evt)
		}
	}
	for _, ovw := range rl.OverwriteResponseEvent {
		ovw(ctx, evt)
	}
	rl.notifyListeners(evt)
	return nil
}

// ServeHTTP implements http.Handler interface.
func (rl *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if rl.ServiceURL == "" {
		rl.ServiceURL = getServiceBaseURL(r)
	}
	if r.Header.Get("Upgrade") == "websocket" {
		rl.HandleWebsocket(w, r)
	} else if r.Header.Get("Accept") == "application/nostr+json" {
		cors.AllowAll().
			Handler(http.HandlerFunc(rl.HandleNIP11)).ServeHTTP(w, r)
	} else {
		rl.serveMux.ServeHTTP(w, r)
	}
}

func (rl *Server) handleDeleteRequest(ctx context.Context,
	evt *nip1.Event) (e error) {

	// event deletion -- nip09
	for _, tag := range evt.Tags {
		if len(tag) >= 2 && tag[0] == "e" {
			// first we fetch the event
			for _, query := range rl.QueryEvents {
				ch, err := query(ctx, nip1.Filter{IDs: []string{tag[1]}})
				if err != nil {
					continue
				}
				target := <-ch
				if target == nil {
					continue
				}
				// got the event, now check if the user can delete it
				acceptDeletion := target.PubKey == evt.PubKey
				var msg string
				if acceptDeletion == false {
					msg = "you are not the author of this event"
				}
				// but if we have a function to overwrite this outcome, use that instead
				for _, odo := range rl.OverwriteDeletionOutcome {
					acceptDeletion, msg = odo(ctx, target, evt)
				}
				if acceptDeletion {
					// delete it
					for _, del := range rl.DeleteEvent {
						rl.Log.D.Chk(del(ctx, target))
					}
				} else {
					// fail and stop here
					return fmt.Errorf("blocked: %s", msg)
				}
				// don't try to query this same event again
				break
			}
		}
	}

	return nil
}

func (rl *Server) HandleWebsocket(w http.ResponseWriter, r *http.Request) {

	conn, e := rl.upgrader.Upgrade(w, r, nil)
	if e != nil {
		rl.Log.E.F("failed to upgrade websocket: %v\n", e)
		return
	}
	rl.clients.Store(conn, struct{}{})
	ticker := time.NewTicker(rl.PingPeriod)
	// NIP-42 challenge
	challenge := make([]byte, 8)
	_, e = rand.Read(challenge)
	if rl.Log.D.Chk(e) {
	}
	ws := &WebSocket{
		conn:      conn,
		Request:   r,
		Challenge: hex.EncodeToString(challenge),
		Authed:    make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(
		context.WithValue(context.Background(), WS_KEY, ws),
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
			rl.removeListener(ws)
		}
	}
	go func() {
		defer kill()
		conn.SetReadLimit(rl.MaxMessageSize)
		e := conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		if rl.Log.D.Chk(e) {
		}
		conn.SetPongHandler(func(string) (e error) {
			return conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		})
		for _, onConnect := range rl.OnConnect {
			onConnect(ctx)
		}
		for {
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
					rl.Log.E.F("unexpected close error from %s: %v\n",
						r.Header.Get("X-Forwarded-For"), e)
				}
				return
			}
			if typ == websocket.PingMessage {
				e = ws.Write(nil)
				continue
			}
			go func(message []byte) {
				envelope, _, _, e := nip1.ProcessEnvelope(message)
				if envelope == nil || rl.Log.D.Chk(e) {
					// stop silently
					return
				}
				var oe *nip1.OKEnvelope
				switch env := envelope.(type) {
				case *nip1.EventEnvelope:
					// check id
					id := env.Event.GetID()
					if id != env.Event.ID {
						oe, e = nip1.NewOKEnvelope(env.Event.ID, false,
							nip1.OKMessage(nip1.OKInvalid,
								"id is computed incorrectly"))
						rl.Log.D.Chk(e)
						rl.Log.D.Chk(ws.Write(oe.Bytes()))
						return
					}
					// check signature
					var ok bool
					if ok, e = env.Event.CheckSignature(); rl.Log.D.Chk(e) {
						oe, e = nip1.NewOKEnvelope(env.Event.ID, false,
							nip1.OKMessage(nip1.OKError,
								"failed to verify signature"))
						rl.Log.D.Chk(e)
						rl.Log.D.Chk(ws.Write(oe.Bytes()))
						return
					} else if !ok {
						oe, e = nip1.NewOKEnvelope(env.Event.ID, false,
							nip1.OKMessage(nip1.OKInvalid,
								"signature is invalid"))
						rl.Log.D.Chk(e)
						rl.Log.D.Chk(ws.Write(oe.Bytes()))
						return
					}
					var writeErr error
					if env.Event.Kind == kind.Deletion {
						// this always returns "blocked: " whenever it returns an error
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
							rl.Log.D.Chk(ws.Write(nip42.NewChallenge(ws.Challenge).Bytes()))
						}
					}
					oe, e = nip1.NewOKEnvelope(env.Event.ID, ok, reason)
					rl.Log.D.Chk(e)
					rl.Log.D.Chk(ws.Write(oe.Bytes()))
				case *nip45.CountRequestEnvelope:
					if rl.CountEvents == nil {
						ce := nip1.NewClosedEnvelope(env.SubscriptionID,
							"unsupported: this relay does not support NIP-45")
						rl.Log.D.Chk(ws.Write(ce.Bytes()))
						return
					}
					var total int64
					for _, filter := range env.Filters {
						total += rl.handleCountRequest(ctx, ws, filter)
					}
					// todo: approximate is a stupid idea
					rl.Log.D.Chk(ws.Write(nip45.
						NewCountResponseEnvelope(env.SubscriptionID, total,
							false).Bytes()))
				case *nip1.ReqEnvelope:
					eose := sync.WaitGroup{}
					eose.Add(len(env.Filters))

					// a context just for the "stored events" request handler
					reqCtx, cancelReqCtx := context.WithCancelCause(ctx)

					// handle each filter separately -- dispatching events as
					// they're loaded from databases
					for _, filter := range env.Filters {
						e = rl.handleRequest(reqCtx, env.SubscriptionID,
							&eose, ws, filter)
						if e != nil {
							// fail everything if any filter is rejected
							reason := e.Error()
							if strings.HasPrefix(reason, "auth-required:") {
								e = ws.Write((&nip42.AuthChallengeEnvelope{Challenge: ws.Challenge}).Bytes())
							}

							rl.Log.D.Chk(ws.Write(nip1.NewClosedEnvelope(
								env.SubscriptionID, reason).Bytes()))
							cancelReqCtx(errors.New("filter rejected"))
							return
						}
					}
					go func() {
						// when all events have been loaded from databases and dispatched
						// we can cancel the context and fire the EOSE message
						eose.Wait()
						cancelReqCtx(nil)
						rl.Log.D.Chk(ws.Write((&nip1.
							EOSEEnvelope{SubscriptionID: env.SubscriptionID}).Bytes()))
					}()
					rl.setListener(env.SubscriptionID, ws, env.Filters,
						cancelReqCtx)
				case *nip1.CloseEnvelope:
					rl.removeListenerId(ws, env.SubscriptionID)
				case *nip42.AuthResponseEnvelope:
					wsBaseUrl := strings.Replace(rl.ServiceURL, "http", "ws", 1)
					if pubkey, ok := nip42.ValidateAuthEvent(env.Event,
						ws.Challenge, wsBaseUrl); ok {
						ws.AuthedPublicKey = pubkey
						close(ws.Authed)
						oe, e = nip1.NewOKEnvelope(env.Event.ID, true, "")
						rl.Log.D.Chk(e)
						rl.Log.D.Chk(ws.Write(oe.Bytes()))
					} else {
						oe, e = nip1.NewOKEnvelope(
							env.Event.ID,
							false,
							nip1.OKMessage(nip1.OKError,
								"failed to authenticate"))
						rl.Log.D.Chk(e)
						rl.Log.D.Chk(ws.Write(oe.Bytes()))
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
				err := ws.Write(nil)
				if err != nil {
					if !strings.HasSuffix(err.Error(),
						"use of closed network connection") {
						rl.Log.W.F(
							"error writing ping: %v; closing websocket\n", err)
					}
					return
				}
			}
		}
	}()
}

func (rl *Server) HandleNIP11(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/nostr+json")
	info := *rl.Info
	for _, ovw := range rl.OverwriteRelayInformation {
		info = ovw(r.Context(), r, info)
	}
	json.NewEncoder(w).Encode(info)
}

func (rl *Server) handleCountRequest(ctx context.Context, ws *WebSocket,
	filter nip1.Filter) int64 {
	// overwrite the filter (for example, to eliminate some kinds or tags that
	// we know we don't support)
	for _, ovw := range rl.OverwriteCountFilter {
		ovw(ctx, &filter)
	}
	var e error
	// then check if we'll reject this filter
	for _, reject := range rl.RejectCountFilter {
		if rejecting, msg := reject(ctx, filter); rejecting {
			e = ws.Write(nip1.NewNoticeEnvelope(msg).Bytes())
			rl.Log.D.Chk(e)
			return 0
		}
	}
	// run the functions to count (generally it will be just one)
	var subtotal int64 = 0
	for _, count := range rl.CountEvents {
		var res int64
		res, e = count(ctx, filter)
		if e != nil {
			e = ws.Write(nip1.NewNoticeEnvelope(e.Error()).Bytes())
			rl.Log.D.Chk(e)
		}
		subtotal += res
	}
	return subtotal
}

func (rl *Server) handleRequest(ctx context.Context, id nip1.SubscriptionID,
	eose *sync.WaitGroup, ws *WebSocket, filter nip1.Filter) (e error) {

	defer eose.Done()
	// overwrite the filter (for example, to eliminate some kinds or that we
	// know we don't support)
	for _, ovw := range rl.OverwriteFilter {
		ovw(ctx, &filter)
	}
	if filter.Limit < 0 {
		return errors.New(nip1.OKMessage(nip1.OKBlocked, "filter invalidated"))
	}
	// then check if we'll reject this filter (we apply this after overwriting
	// because we may, for example, remove some things from the incoming filters
	// that we know we don't support, and then if the end result is an empty
	// filter we can just reject it)
	for _, rejectFilter := range rl.RejectFilter {
		if reject, msg := rejectFilter(ctx, filter); reject {
			e = ws.Write(nip1.NewNoticeEnvelope(msg).Bytes())
			return errors.New(nip1.OKMessage(nip1.OKBlocked, msg))
		}
	}
	// run the functions to query events (generally just one, but we might be
	// fetching stuff from multiple places)
	eose.Add(len(rl.QueryEvents))
	for _, query := range rl.QueryEvents {
		var ch chan *nip1.Event
		ch, e = query(ctx, filter)
		if e != nil {
			rl.Log.D.Chk(ws.Write(nip1.NewNoticeEnvelope(e.Error()).Bytes()))
			eose.Done()
			continue
		}
		go func(ch chan *nip1.Event) {
			for event := range ch {
				for _, ovw := range rl.OverwriteResponseEvent {
					ovw(ctx, event)
				}
				rl.Log.D.Chk(ws.Write((&nip1.EventEnvelope{
					SubscriptionID: id,
					Event:          event,
				}).Bytes()))
			}
			eose.Done()
		}(ch)
	}
	return nil
}
