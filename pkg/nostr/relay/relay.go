package relay

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/enveloper"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/connect"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/OK"
	auth2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countresponse"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eose"
	event2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/puzpuzpuz/xsync/v2"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
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
type WithAuthHandler func(c context.T, authEvent *event.T) (ok bool)

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
	ctx                           context.T // will be canceled when the connection closes
	cancel                        context.F
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
func New(c context.T, url string, opts ...Option) (r *Relay) {
	ctx, cancel := context.Cancel(c)
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
						if status, e = r.Auth(r.ctx, authEvent); log.Fail(e) {
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
func RelayConnect(c context.T, url string,
	opts ...Option) (r *Relay, e error) {

	r = New(context.Bg(), url, opts...)
	e = r.Connect(c)
	return
}

// When instantiating relay connections, some options may be passed.

// String just returns the relay URL.
func (r *Relay) String() string { return r.URL }

// Context retrieves the context that is associated with this relay connection.
func (r *Relay) Context() context.T { return r.ctx }

// IsConnected returns true if the connection to this relay seems to be active.
func (r *Relay) IsConnected() bool { return r.ctx.Err() == nil }

func (r *Relay) queued(ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			if e := wsutil.WriteClientMessage(r.Connection.Conn, ws.OpPing,
				nil); log.Fail(e) {
				log.E.F("{%s} error writing ping: %v; closing websocket",
					r.URL, e)
				if e = r.Close(); log.Fail(e) {
				} // this should trigger a context cancelation
				return
			}
		case writeReq := <-r.writeQueue:
			// all write requests will go through this to prevent races
			if e := r.Connection.WriteMessage(writeReq.Msg); log.Fail(e) {
				writeReq.Answer <- e
			}
			close(writeReq.Answer)
		case <-r.ctx.Done():
			// stop here
			return
		}
	}
}

func (r *Relay) closer(ticker *time.Ticker) {
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
}

func (r *Relay) reader(conn *connect.Connection) {
	var e error
	buf := new(bytes.Buffer)
	for {
		buf.Reset()
		if e := conn.ReadMessage(r.ctx, buf); log.Fail(e) {
			r.Err = e
			log.D.Chk(r.Close())
			break
		}
		message := buf.Bytes()
		log.D.F("{%s} %v", r.URL, string(message))
		var envelope enveloper.I
		envelope, _, e = envelopes.ProcessEnvelope(message)
		if envelope == nil || log.Fail(e) {
			continue
		}

		switch env := envelope.(type) {
		case *notice.Envelope:
			// see WithNoticeHandler
			if r.notices != nil {
				r.notices <- env.Text
			} else {
				log.D.F("NOTICE from %s: '%s'", r.URL, env.Text)
			}
		case *auth2.Challenge:
			if env.Challenge == "" {
				continue
			}
			// see WithAuthHandler
			if r.challenges != nil {
				r.challenges <- env.Challenge
			}
		case *event2.Envelope:
			if env.SubscriptionID == "" {
				continue
			}
			if subscr, ok := r.Subscriptions.Load(string(env.SubscriptionID)); !ok {
				log.D.F("{%s} no subscr with id '%s'", r.URL,
					env.SubscriptionID)
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
						if log.Fail(e) {
							msg = e.Error()
						}
						log.E.F("{%s} bad signature: %s", r.URL, msg)
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
}

// Connect tries to establish a websocket connection to r.URL. If the context
// expires before the connection is complete, an error is returned. Once
// successfully connected, context expiration has no effect: call r.Close to
// close the connection.
//
// The underlying relay connection will use a background context. If you want to
// pass a custom context to the underlying relay connection, use New() and then
// Relay.Connect().
func (r *Relay) Connect(c context.T) (e error) {
	if r.ctx == nil || r.Subscriptions == nil {
		return fmt.Errorf("relay must be initialized with a call to New()")
	}
	if r.URL == "" {
		return fmt.Errorf("invalid relay URL '%s'", r.URL)
	}
	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, set it to 7 seconds
		var cancel context.F
		c, cancel = context.Timeout(c, 7*time.Second)
		defer cancel()
	}
	var conn *connect.Connection
	if conn, e = connect.NewConnection(c, r.URL, r.RequestHeader); log.Fail(e) {
		return fmt.Errorf("error opening websocket to '%s': %w", r.URL, e)
	}
	r.Connection = conn
	// ping every 29 seconds
	ticker := time.NewTicker(29 * time.Second)
	// to be used when the connection is closed
	go r.closer(ticker)
	go r.queued(ticker)
	go r.reader(conn)
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
func (r *Relay) Publish(c context.T, evt *event.T) (s Status, e error) {
	s = PublishStatusFailed
	// data races on status variable without this mutex
	var mu sync.Mutex
	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		c, cancel = context.Timeout(c, 7*time.Second)
		defer cancel()
	}
	// make it cancellable so we can stop everything upon receiving an "OK"
	var cancel context.F
	c, cancel = context.Cancel(c)
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
			e = fmt.Errorf("publish failed: %s", reason)
		}
		cancel()
	}
	r.okCallbacks.Store(string(evt.ID), okCallback)
	defer r.okCallbacks.Delete(string(evt.ID))
	// publish event
	envb, _ := (&event2.Envelope{Event: evt}).MarshalJSON()
	log.D.F("{%s} sending %v", r.URL, string(envb))
	s = PublishStatusSent
	if e = <-r.Write(envb); log.Fail(e) {
		s = PublishStatusFailed
		return
	}
	for {
		select {
		case <-c.Done(): // this will be called when we get an OK
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
func (r *Relay) Auth(c context.T, event *event.T) (s Status, e error) {
	s = PublishStatusFailed
	// data races on s variable without this mutex
	var mu sync.Mutex
	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 3 seconds
		var cancel context.F
		c, cancel = context.Timeout(c, 3*time.Second)
		defer cancel()
	}
	// make it cancellable so we can stop everything upon receiving an "OK"
	var cancel context.F
	c, cancel = context.Cancel(c)
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
	authResponse, _ := (&auth2.Response{T: event}).MarshalJSON()
	log.D.F("{%s} sending %v", r.URL, string(authResponse))
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
	<-c.Done()
	mu.Lock()
	defer mu.Unlock()
	return s, e
}

// Subscribe sends a "REQ" command to the relay r as in NIP-01. Events are
// returned through the channel sub.Events. The subscription is closed when
// context ctx is cancelled ("CLOSE" in NIP-01).
//
// Remember to Cancel subscriptions, either by calling `.Unsub()` on them or
// ensuring their `context.T` will be canceled at some point. Failure to
// do that will result in a huge number of halted goroutines being created.
func (r *Relay) Subscribe(c context.T, filters filters.T,
	opts ...SubscriptionOption) (s *Subscription, e error) {

	s = r.PrepareSubscription(c, filters, opts...)
	if e = s.Fire(); log.Fail(e) {
		return nil, fmt.Errorf("couldn't subscribe to %v at %s: %w",
			filters, r.URL, e)
	}
	return
}

// PrepareSubscription creates a subscription, but doesn't fire it.
//
// Remember to Cancel subscriptions, either by calling `.Unsub()` on them or
// ensuring their `context.T` will be canceled at some point. Failure to
// do that will result in a huge number of halted goroutines being created.
func (r *Relay) PrepareSubscription(c context.T, filters filters.T,
	opts ...SubscriptionOption) (s *Subscription) {

	if r.Connection == nil {
		panic(fmt.
			Errorf("must call .Connect() first before calling .Subscribe()"))
	}
	current := r.subscriptionIDCounter.Add(1)
	ctx, cancel := context.Cancel(c)
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

func (r *Relay) QuerySync(c context.T, f *filter.T,
	opts ...SubscriptionOption) (evs []*event.T, e error) {

	var sub *Subscription
	if sub, e = r.Subscribe(c, filters.T{f}, opts...); log.Fail(e) {
		return
	}
	defer sub.Unsub()
	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		c, cancel = context.Timeout(c, 7*time.Second)
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
		case <-c.Done():
			return
		}
	}
}

func (r *Relay) Count(c context.T, filters filters.T,
	opts ...SubscriptionOption) (cnt int64, e error) {

	sub := r.PrepareSubscription(c, filters, opts...)
	sub.CountResult = make(chan int64)
	if e = sub.Fire(); log.Fail(e) {
		return
	}
	defer sub.Unsub()
	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		c, cancel = context.Timeout(c, 7*time.Second)
		defer cancel()
	}
	for {
		select {
		case cnt = <-sub.CountResult:
			return
		case <-c.Done():
			return 0, c.Err()
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
