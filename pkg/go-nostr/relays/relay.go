package relays

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/OK"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/closed"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/connection"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/count"
	envelope2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelope"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/puzpuzpuz/xsync/v2"
)

type Status int

var subscriptionIDCounter atomic.Int32

type Relay struct {
	closeMutex sync.Mutex

	URL           string
	RequestHeader http.Header // e.g. for origin header

	Connection    *connection.Connection
	Subscriptions *xsync.MapOf[string, *Subscription]

	ConnectionError         error
	connectionContext       context.T // will be canceled when the connection closes
	connectionContextCancel context.F

	challenge                     string      // NIP-42 challenge, we only keep the last
	notices                       chan string // NIP-01 NOTICEs
	okCallbacks                   *xsync.MapOf[string, func(bool, string)]
	writeQueue                    chan writeRequest
	subscriptionChannelCloseQueue chan *Subscription

	// custom things that aren't often used
	//
	AssumeValid bool // this will skip verifying signatures for events received from this relay
}

type writeRequest struct {
	msg    []byte
	answer chan error
}

// NewRelay returns a new relay. The relay connection will be closed when the context is canceled.
func NewRelay(c context.T, url string, opts ...RelayOption) *Relay {
	ctx, cancel := context.Cancel(c)
	r := &Relay{
		URL:                           normalize.URL(url),
		connectionContext:             ctx,
		connectionContextCancel:       cancel,
		Subscriptions:                 xsync.NewMapOf[*Subscription](),
		okCallbacks:                   xsync.NewMapOf[func(bool, string)](),
		writeQueue:                    make(chan writeRequest),
		subscriptionChannelCloseQueue: make(chan *Subscription),
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case WithNoticeHandler:
			r.notices = make(chan string)
			go func() {
				for n := range r.notices {
					o(n)
				}
			}()
		}
	}

	return r
}

// RelayConnect returns a relay object connected to url.
// Once successfully connected, cancelling ctx has no effect.
// To close the connection, call r.Close().
func RelayConnect(c context.T, url string, opts ...RelayOption) (*Relay, error) {
	r := NewRelay(context.Bg(), url, opts...)
	e := r.Connect(c)
	return r, e
}

// When instantiating relay connections, some options may be passed.

// RelayOption is the type of the argument passed for that.
type RelayOption interface {
	IsRelayOption()
}

// WithNoticeHandler just takes notices and is expected to do something with them.
// when not given, defaults to logging the notices.
type WithNoticeHandler func(notice string)

func (_ WithNoticeHandler) IsRelayOption() {}

var _ RelayOption = (WithNoticeHandler)(nil)

// String just returns the relay URL.
func (r *Relay) String() string {
	return r.URL
}

// Context retrieves the context that is associated with this relay connection.
func (r *Relay) Context() context.T { return r.connectionContext }

// IsConnected returns true if the connection to this relay seems to be active.
func (r *Relay) IsConnected() bool { return r.connectionContext.Err() == nil }

// Connect tries to establish a websocket connection to r.URL.
// If the context expires before the connection is complete, an error is returned.
// Once successfully connected, context expiration has no effect: call r.Close
// to close the connection.
//
// The underlying relay connection will use a background context. If you want to
// pass a custom context to the underlying relay connection, use NewRelay() and
// then Relay.Connect().
func (r *Relay) Connect(c context.T) error {
	if r.connectionContext == nil || r.Subscriptions == nil {
		return fmt.Errorf("relay must be initialized with a call to NewRelay()")
	}

	if r.URL == "" {
		return fmt.Errorf("invalid relay URL '%s'", r.URL)
	}

	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		c, cancel = context.Timeout(c, 7*time.Second)
		defer cancel()
	}

	conn, e := connection.NewConnection(c, r.URL, r.RequestHeader)
	if e != nil {
		return fmt.Errorf("error opening websocket to '%s': %w", r.URL, e)
	}
	r.Connection = conn

	// ping every 29 seconds
	ticker := time.NewTicker(29 * time.Second)

	// to be used when the connection is closed
	go func() {
		<-r.connectionContext.Done()
		// close these things when the connection is closed
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
				e := wsutil.WriteClientMessage(r.Connection.Conn, ws.OpPing, nil)
				if e != nil {
					fmt.Printf("{%s} error writing ping: %v; closing websocket", r.URL, e)
					r.Close() // this should trigger a context cancelation
					return
				}
			case writeRequest := <-r.writeQueue:
				// all write requests will go through this to prevent races
				if e := r.Connection.WriteMessage(writeRequest.msg); e != nil {
					writeRequest.answer <- e
				}
				close(writeRequest.answer)
			case <-r.connectionContext.Done():
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
			if e := conn.ReadMessage(r.connectionContext, buf); e != nil {
				r.ConnectionError = e
				r.Close()
				break
			}

			message := buf.Bytes()
			fmt.Printf("{%s} %v\n", r.URL, string(message))
			envelope := envelope2.ParseMessage(message)
			if envelope == nil {
				continue
			}

			switch env := envelope.(type) {
			case *notice.Envelope:
				// see WithNoticeHandler
				if r.notices != nil {
					r.notices <- string(*env)
				} else {
					log.Printf("NOTICE from %s: '%s'\n", r.URL, string(*env))
				}
			case *auth.Envelope:
				if env.Challenge == nil {
					continue
				}
				r.challenge = *env.Challenge
			case *event.Envelope:
				if env.SubscriptionID == nil {
					continue
				}
				if subscription, ok := r.Subscriptions.Load(*env.SubscriptionID); !ok {
					fmt.Printf("{%s} no subscription with id '%s'\n", r.URL, *env.SubscriptionID)
					continue
				} else {
					// check if the event matches the desired filter, ignore otherwise
					if !subscription.Filters.Match(&env.T) {
						fmt.Printf("{%s} filter does not match: %v ~ %v\n", r.URL, subscription.Filters, env.T)
						continue
					}

					// check signature, ignore invalid, except from trusted (AssumeValid) relays
					if !r.AssumeValid {
						if ok, e := env.T.CheckSignature(); !ok {
							errmsg := ""
							if e != nil {
								errmsg = e.Error()
							}
							fmt.Printf("{%s} bad signature on %s; %s\n", r.URL, env.T.ID, errmsg)
							continue
						}
					}

					// dispatch this to the internal .events channel of the subscription
					subscription.dispatchEvent(&env.T)
				}
			case *eose.Envelope:
				if subscription, ok := r.Subscriptions.Load(string(*env)); ok {
					subscription.dispatchEose()
				}
			case *closed.Envelope:
				if subscription, ok := r.Subscriptions.Load(string(env.SubscriptionID)); ok {
					subscription.dispatchClosed(env.Reason)
				}
			case *count.Envelope:
				if subscription, ok := r.Subscriptions.Load(string(env.SubscriptionID)); ok && env.Count != nil && subscription.countResult != nil {
					subscription.countResult <- *env.Count
				}
			case *OK.Envelope:
				if okCallback, exist := r.okCallbacks.Load(env.EventID); exist {
					okCallback(env.OK, env.Reason)
				} else {
					fmt.Printf("{%s} got an unexpected OK message for event %s", r.URL, env.EventID)
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
	case r.writeQueue <- writeRequest{msg: msg, answer: ch}:
	case <-r.connectionContext.Done():
		go func() { ch <- fmt.Errorf("connection closed") }()
	}
	return ch
}

// Publish sends an "EVENT" command to the relay r as in NIP-01 and waits for an OK response.
func (r *Relay) Publish(c context.T, ev event.T) error {
	return r.publish(c, ev.ID, &event.Envelope{T: ev})
}

// Auth sends an "AUTH" command client->relay as in NIP-42 and waits for an OK response.
func (r *Relay) Auth(c context.T, sign func(ev *event.T) error) error {
	authEvent := event.T{
		CreatedAt: timestamp.Now(),
		Kind:      event.KindClientAuthentication,
		Tags: tags.Tags{
			tags.Tag{"relay", r.URL},
			tags.Tag{"challenge", r.challenge},
		},
		Content: "",
	}
	if e := sign(&authEvent); e != nil {
		return fmt.Errorf("error signing auth event: %w", e)
	}

	return r.publish(c, authEvent.ID, &auth.Envelope{Event: authEvent})
}

// publish can be used both for EVENT and for AUTH
func (r *Relay) publish(c context.T, id string, env envelopes.E) error {
	var e error
	var cancel context.F

	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		c, cancel = context.TimeoutCause(c, 7*time.Second, fmt.Errorf("given up waiting for an OK"))
		defer cancel()
	} else {
		// otherwise make the context cancellable so we can stop everything upon receiving an "OK"
		c, cancel = context.Cancel(c)
		defer cancel()
	}

	// listen for an OK callback
	gotOk := false
	r.okCallbacks.Store(id, func(ok bool, reason string) {
		gotOk = true
		if !ok {
			e = fmt.Errorf("msg: %s", reason)
		}
		cancel()
	})
	defer r.okCallbacks.Delete(id)

	// publish event
	envb, _ := env.MarshalJSON()
	fmt.Printf("{%s} sending %v\n", r.URL, string(envb))
	if e := <-r.Write(envb); e != nil {
		return e
	}

	for {
		select {
		case <-c.Done():
			// this will be called when we get an OK or when the context has been canceled
			if gotOk {
				return e
			}
			return c.Err()
		case <-r.connectionContext.Done():
			// this is caused when we lose connectivity
			return e
		}
	}
}

// Subscribe sends a "REQ" command to the relay r as in NIP-01.
// Events are returned through the channel sub.Events.
// The subscription is closed when context ctx is cancelled ("CLOSE" in NIP-01).
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or ensuring their `context.T` will be canceled at some point.
// Failure to do that will result in a huge number of halted goroutines being created.
func (r *Relay) Subscribe(c context.T, filters filters.T, opts ...SubscriptionOption) (*Subscription, error) {
	sub := r.PrepareSubscription(c, filters, opts...)

	if e := sub.Fire(); e != nil {
		return nil, fmt.Errorf("couldn't subscribe to %v at %s: %w", filters, r.URL, e)
	}

	return sub, nil
}

// PrepareSubscription creates a subscription, but doesn't fire it.
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or ensuring their `context.T` will be canceled at some point.
// Failure to do that will result in a huge number of halted goroutines being created.
func (r *Relay) PrepareSubscription(c context.T, filters filters.T, opts ...SubscriptionOption) *Subscription {
	if r.Connection == nil {
		panic(fmt.Errorf("must call .Connect() first before calling .Subscribe()"))
	}

	current := subscriptionIDCounter.Add(1)
	ctx, cancel := context.Cancel(c)

	sub := &Subscription{
		Relay:             r,
		Context:           ctx,
		cancel:            cancel,
		counter:           int(current),
		Events:            make(chan *event.T),
		EndOfStoredEvents: make(chan struct{}),
		ClosedReason:      make(chan string, 1),
		Filters:           filters,
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case WithLabel:
			sub.label = string(o)
		}
	}

	id := sub.GetID()
	r.Subscriptions.Store(id, sub)

	// start handling events, eose, unsub etc:
	go sub.start()

	return sub
}

func (r *Relay) QuerySync(c context.T, f filter.T, opts ...SubscriptionOption) ([]*event.T, error) {
	sub, e := r.Subscribe(c, filters.T{f}, opts...)
	if e != nil {
		return nil, e
	}

	defer sub.Unsub()

	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		c, cancel = context.Timeout(c, 7*time.Second)
		defer cancel()
	}

	var events []*event.T
	for {
		select {
		case evt := <-sub.Events:
			if evt == nil {
				// channel is closed
				return events, nil
			}
			events = append(events, evt)
		case <-sub.EndOfStoredEvents:
			return events, nil
		case <-c.Done():
			return events, nil
		}
	}
}

func (r *Relay) Count(c context.T, filters filters.T, opts ...SubscriptionOption) (int64, error) {
	sub := r.PrepareSubscription(c, filters, opts...)
	sub.countResult = make(chan int64)

	if e := sub.Fire(); e != nil {
		return 0, e
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
		case count := <-sub.countResult:
			return count, nil
		case <-c.Done():
			return 0, c.Err()
		}
	}
}

func (r *Relay) Close() error {
	r.closeMutex.Lock()
	defer r.closeMutex.Unlock()

	if r.connectionContextCancel == nil {
		return fmt.Errorf("relay not connected")
	}

	r.connectionContextCancel()
	r.connectionContextCancel = nil
	return r.Connection.Close()
}
