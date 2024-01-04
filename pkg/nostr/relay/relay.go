package relay

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/connect"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/OK"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/countresponse"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/puzpuzpuz/xsync/v2"
)

// Option is the type of the argument passed for that. Some examples of this are
// WithNoticeHandler and WithAuthHandler.
type Option interface {
	IsRelayOption()
}

// WithNoticeHandler just takes notices and is expected to do something with
// them. when not given, defaults to logging the notices.
type WithNoticeHandler func(notice string)

func (_ WithNoticeHandler) IsRelayOption() {}

// compile time interface check
var _ Option = (WithNoticeHandler)(nil)

// WithAuthHandler takes an auth event and expects it to be signed. when not
// given, AUTH messages from relays are ignored.
type WithAuthHandler func(ctx context.Context, authEvent *event.T) (ok bool)

func (_ WithAuthHandler) IsRelayOption() {}

// compile time interface check
var _ Option = (WithAuthHandler)(nil)

type Status int

const (
	PublishStatusSent      Status = 0
	PublishStatusFailed    Status = -1
	PublishStatusSucceeded Status = 1
)

func (s Status) String() string {
	switch s {
	case PublishStatusSent:
		return "sent"
	case PublishStatusFailed:
		return "failed"
	case PublishStatusSucceeded:
		return "success"
	}
	return "unknown"
}

type Relay struct {
	URL                           string
	RequestHeader                 http.Header // e.g. for origin header
	Connection                    *connect.Connection
	Subscriptions                 *xsync.MapOf[string, *Subscription]
	Err                           error
	ctx                           context.Context // will be canceled when the connection closes
	cancel                        context.CancelFunc
	challenges                    chan string // NIP-42 challenges
	notices                       chan string // NIP-01 NOTICEs
	okCallbacks                   *xsync.MapOf[string, func(bool, *string)]
	writeQueue                    chan WriteRequest
	subscriptionChannelCloseQueue chan *Subscription
	subscriptionIDCounter         atomic.Int32
	// custom things that aren't often used
	//
	AssumeValid bool // this will skip verifying signatures for events received from this relay
}

type WriteRequest struct {
	Msg    []byte
	Answer chan error
}

// New returns a new relay. The relay connection will be closed when the context
// is canceled.
func New(ctx context.Context, url string, opts ...Option) (r *Relay) {
	ctx, cancel := context.WithCancel(ctx)
	r = &Relay{
		URL:                           normalize.URL(url),
		ctx:                           ctx,
		cancel:                        cancel,
		Subscriptions:                 xsync.NewMapOf[*Subscription](),
		okCallbacks:                   xsync.NewMapOf[func(bool, *string)](),
		writeQueue:                    make(chan WriteRequest),
		subscriptionChannelCloseQueue: make(chan *Subscription),
	}
	for _, opt := range opts {
		switch o := opt.(type) {
		case WithNoticeHandler:
			r.notices = make(chan string)
			go func() {
				for notice := range r.notices {
					o(notice)
				}
			}()
		case WithAuthHandler:
			r.challenges = make(chan string)
			go func() {
				for challenge := range r.challenges {
					authEvent := &event.T{
						CreatedAt: timestamp.Now(),
						Kind:      kind.ClientAuthentication,
						Tags: tags.T{
							{"relay", url},
							{"challenge", challenge},
						},
						Content: "",
					}
					var e error
					var status Status
					if ok := o(r.ctx, authEvent); ok {
						if status, e = r.Auth(r.ctx, authEvent); fails(e) {
						}
						log.D.Ln(status.String())
					}
				}
			}()
		}
	}
	return r
}

// RelayConnect returns a relay object connected to url. Once successfully
// connected, cancelling ctx has no effect. To close the connection, call
// r.Close().
func RelayConnect(ctx context.Context, url string,
	opts ...Option) (r *Relay, e error) {

	r = New(context.Background(), url, opts...)
	e = r.Connect(ctx)
	return
}

// When instantiating relay connections, some options may be passed.

// String just returns the relay URL.
func (r *Relay) String() string { return r.URL }

// Context retrieves the context that is associated with this relay connection.
func (r *Relay) Context() context.Context { return r.ctx }

// IsConnected returns true if the connection to this relay seems to be active.
func (r *Relay) IsConnected() bool { return r.ctx.Err() == nil }

// Connect tries to establish a websocket connection to r.URL. If the context
// expires before the connection is complete, an error is returned. Once
// successfully connected, context expiration has no effect: call r.Close to
// close the connection.
//
// The underlying relay connection will use a background context. If you want to
// pass a custom context to the underlying relay connection, use New() and then
// Relay.Connect().
func (r *Relay) Connect(ctx context.Context) (e error) {
	if r.ctx == nil || r.Subscriptions == nil {
		return fmt.Errorf("relay must be initialized with a call to New()")
	}
	if r.URL == "" {
		return fmt.Errorf("invalid relay URL '%s'", r.URL)
	}
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, set it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}
	var conn *connect.Connection
	if conn, e = connect.NewConnection(ctx, r.URL, r.RequestHeader); fails(e) {
		return fmt.Errorf("error opening websocket to '%s': %w", r.URL, e)
	}
	r.Connection = conn
	// ping every 29 seconds
	ticker := time.NewTicker(29 * time.Second)
	// to be used when the connection is closed
	go func() {
		<-r.ctx.Done()
		// close these things when the connection is closed
		if r.challenges != nil {
			close(r.challenges)
		}
		if r.notices != nil {
			close(r.notices)
		}
		// stop the ticker
		ticker.Stop()
		// close all subscriptions
		r.Subscriptions.Range(func(_ string, sub *Subscription) bool {
			go sub.Unsub()
			return true
		})
	}()

	// queue all write operations here so we don't do mutex spaghetti
	go func() {
		for {
			select {
			case <-ticker.C:
				if e := wsutil.WriteClientMessage(r.Connection.Conn, ws.OpPing, nil); fails(e) {
					log.E.F("{%s} error writing ping: %v; closing websocket", r.URL, e)
					if e = r.Close(); fails(e) {
					} // this should trigger a context cancelation
					return
				}
			case writeReq := <-r.writeQueue:
				// all write requests will go through this to prevent races
				if e := r.Connection.WriteMessage(writeReq.Msg); fails(e) {
					writeReq.Answer <- e
				}
				close(writeReq.Answer)
			case <-r.ctx.Done():
				// stop here
				return
			}
		}
	}()

	// general message reader loop
	go func() {
		buf := new(bytes.Buffer)
		for {
			buf.Reset()
			if e := conn.ReadMessage(r.ctx, buf); fails(e) {
				r.Err = e
				r.Close()
				break
			}
			message := buf.Bytes()
			log.D.F("{%s} %v", r.URL, string(message))
			var envelope enveloper.Enveloper
			envelope, _, _, e = enveloper.ProcessEnvelope(message)
			if envelope == nil || fails(e) {
				continue
			}

			switch env := envelope.(type) {
			case *notice.Envelope:
				// see WithNoticeHandler
				if r.notices != nil {
					r.notices <- env.Text
				} else {
					log.D.F("NOTICE from %s: '%s'\n", r.URL, env.Text)
				}
			case *auth.ChallengeEnvelope:
				if env.Challenge == "" {
					continue
				}
				// see WithAuthHandler
				if r.challenges != nil {
					r.challenges <- env.Challenge
				}
			case *event.Envelope:
				if env.SubscriptionID == "" {
					continue
				}
				if subscr, ok := r.Subscriptions.Load(string(env.SubscriptionID)); !ok {
					log.D.F("{%s} no subscr with id '%s'", r.URL, env.SubscriptionID)
					continue
				} else {
					// check if the event matches the desired filter, ignore otherwise
					if !subscr.Filters.Match(env.Event) {
						log.E.F("{%s} filter does not match: %v ~ %v",
							r.URL, subscr.Filters, env.Event)
						continue
					}
					// check signature, ignore invalid, except from trusted (AssumeValid) relays
					if !r.AssumeValid {
						if ok, e = env.Event.CheckSignature(); !ok {
							msg := ""
							if fails(e) {
								msg = e.Error()
							}
							log.E.F("{%s} bad signature: %s\n", r.URL, msg)
							continue
						}
					}
					// dispatch this to the internal .events channel of the subscr
					subscr.DispatchEvent(env.Event)
				}
			case *eose.Envelope:
				if sub, ok := r.Subscriptions.Load(string(env.T)); ok {
					sub.DispatchEose()
				}
			case *countresponse.Envelope:
				if sub, ok := r.Subscriptions.Load(string(env.SubscriptionID)); ok &&
					env.Count != 0 &&
					sub.CountResult != nil {
					sub.CountResult <- env.Count
				}
			case *OK.Envelope:
				if okCallback, exist := r.okCallbacks.Load(string(env.EventID)); exist {
					okCallback(env.OK, &env.Reason)
				}
			}
		}
	}()
	return nil
}

// Write queues a message to be sent to the relay.
func (r *Relay) Write(msg []byte) <-chan error {
	ch := make(chan error)
	select {
	case r.writeQueue <- WriteRequest{Msg: msg, Answer: ch}:
	case <-r.ctx.Done():
		go func() { ch <- fmt.Errorf("connection closed") }()
	}
	return ch
}

// Publish sends an "EVENT" command to the relay r as in NIP-01. Status can be:
// success, failed, or sent (no response from relay before ctx times out).
func (r *Relay) Publish(ctx context.Context, evt *event.T) (s Status, e error) {
	s = PublishStatusFailed
	// data races on status variable without this mutex
	var mu sync.Mutex
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}
	// make it cancellable so we can stop everything upon receiving an "OK"
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	// listen for an OK callback
	okCallback := func(ok bool, msg *string) {
		mu.Lock()
		defer mu.Unlock()
		if ok {
			s = PublishStatusSucceeded
		} else {
			s = PublishStatusFailed
			reason := ""
			if msg != nil {
				reason = *msg
			}
			e = fmt.Errorf("Msg: %s", reason)
		}
		cancel()
	}
	r.okCallbacks.Store(string(evt.ID), okCallback)
	defer r.okCallbacks.Delete(string(evt.ID))
	// publish event
	envb, _ := (&event.Envelope{Event: evt}).MarshalJSON()
	log.D.F("{%s} sending %v\n", r.URL, string(envb))
	s = PublishStatusSent
	if e = <-r.Write(envb); fails(e) {
		s = PublishStatusFailed
		return
	}
	for {
		select {
		case <-ctx.Done(): // this will be called when we get an OK
			// proceed to return status as it is e.g. if this happens because of
			// the timeout then status will probably be "failed" but if it
			// happens because okCallback was called then it might be
			// "succeeded" do not return if okCallback is in process
			return
		case <-r.ctx.Done():
			// same as above, but when the relay loses connectivity entirely
			return
		}
	}
}

// Auth sends an "AUTH" command client -> relay as in NIP-42.
//
// Status can be: success, failed, or sent (no response from relay before ctx
// times out).
func (r *Relay) Auth(ctx context.Context, event *event.T) (s Status, e error) {
	s = PublishStatusFailed
	// data races on s variable without this mutex
	var mu sync.Mutex
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 3 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}
	// make it cancellable so we can stop everything upon receiving an "OK"
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	// listen for an OK callback
	okCallback := func(ok bool, msg *string) {
		mu.Lock()
		if ok {
			s = PublishStatusSucceeded
		} else {
			s = PublishStatusFailed
			reason := ""
			if msg != nil {
				reason = *msg
			}
			e = fmt.Errorf("Msg: %s", reason)
		}
		mu.Unlock()
		cancel()
	}
	r.okCallbacks.Store(string(event.ID), okCallback)
	defer r.okCallbacks.Delete(string(event.ID))
	// send AUTH
	authResponse, _ := (&auth.ResponseEnvelope{T: event}).MarshalJSON()
	log.D.F("{%s} sending %v\n", r.URL, string(authResponse))
	if e = <-r.Write(authResponse); e != nil {
		// s will be "failed"
		return s, e
	}
	// use mu.Lock() just in case the okCallback got called, extremely unlikely.
	mu.Lock()
	s = PublishStatusSent
	mu.Unlock()
	// the context either times out, and the s is "sent" or the okCallback is
	// called and the s is set to "succeeded" or "failed" NIP-42 does not
	// mandate an "OK" reply to an "AUTH" message
	<-ctx.Done()
	mu.Lock()
	defer mu.Unlock()
	return s, e
}

// Subscribe sends a "REQ" command to the relay r as in NIP-01. Events are
// returned through the channel sub.Events. The subscription is closed when
// context ctx is cancelled ("CLOSE" in NIP-01).
//
// Remember to Cancel subscriptions, either by calling `.Unsub()` on them or
// ensuring their `context.Context` will be canceled at some point. Failure to
// do that will result in a huge number of halted goroutines being created.
func (r *Relay) Subscribe(ctx context.Context, filters filters.T,
	opts ...SubscriptionOption) (s *Subscription, e error) {

	s = r.PrepareSubscription(ctx, filters, opts...)
	if e = s.Fire(); fails(e) {
		return nil, fmt.Errorf("couldn't subscribe to %v at %s: %w", filters, r.URL, e)
	}
	return
}

// PrepareSubscription creates a subscription, but doesn't fire it.
//
// Remember to Cancel subscriptions, either by calling `.Unsub()` on them or
// ensuring their `context.Context` will be canceled at some point. Failure to
// do that will result in a huge number of halted goroutines being created.
func (r *Relay) PrepareSubscription(ctx context.Context, filters filters.T,
	opts ...SubscriptionOption) (s *Subscription) {

	if r.Connection == nil {
		panic(fmt.Errorf("must call .Connect() first before calling .Subscribe()"))
	}
	current := r.subscriptionIDCounter.Add(1)
	ctx, cancel := context.WithCancel(ctx)
	s = &Subscription{
		Relay:             r,
		Context:           ctx,
		Cancel:            cancel,
		Counter:           int(current),
		Events:            make(chan *event.T),
		EndOfStoredEvents: make(chan struct{}),
		Filters:           filters,
	}
	for _, opt := range opts {
		switch o := opt.(type) {
		case WithLabel:
			s.Label = string(o)
		}
	}
	id := s.GetID()
	r.Subscriptions.Store(id, s)
	// Start handling events, eose, unsub etc:
	go s.Start()
	return
}

func (r *Relay) QuerySync(ctx context.Context, filter *filter.T,
	opts ...SubscriptionOption) (evs []*event.T, e error) {

	var sub *Subscription
	if sub, e = r.Subscribe(ctx, filters.T{filter}, opts...); fails(e) {
		return
	}
	defer sub.Unsub()
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}
	for {
		select {
		case ev := <-sub.Events:
			if ev == nil {
				// channel is closed
				return
			}
			evs = append(evs, ev)
		case <-sub.EndOfStoredEvents:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (r *Relay) Count(ctx context.Context, filters filters.T,
	opts ...SubscriptionOption) (c int64, e error) {

	sub := r.PrepareSubscription(ctx, filters, opts...)
	sub.CountResult = make(chan int64)
	if e = sub.Fire(); fails(e) {
		return
	}
	defer sub.Unsub()
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}
	for {
		select {
		case c = <-sub.CountResult:
			return
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
}

func (r *Relay) Close() (e error) {
	if r.cancel == nil {
		return fmt.Errorf("relay not connected")
	}
	r.cancel()
	r.cancel = nil
	return r.Connection.Close()
}
