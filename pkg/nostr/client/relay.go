package client

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/connection"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/authenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closedenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eoseenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/noticeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/subscriptionoption"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayinfo"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscription"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/puzpuzpuz/xsync/v2"
)

var log, chk = slog.New(os.Stderr)

type Status int

var subscriptionIDCounter atomic.Int32

type T struct {
	closeMutex                    sync.Mutex
	url                           string
	RequestHeader                 http.Header // e.g. for origin header
	Connection                    *connection.C
	Subscriptions                 *xsync.MapOf[string, *subscription.T]
	ConnectionError               error
	ConnectionContext             context.T // will be canceled when connection closes
	ConnectionContextCancel       context.F
	done                          sync.Once
	challenge                     string // NIP-42 challenge, only keep the last
	AuthRequired                  chan struct{}
	AuthEventID                   eventid.T
	Authed                        chan struct{}
	notices                       chan string // NIP-01 NOTICEs
	okCallbacks                   *xsync.MapOf[string, func(bool, string)]
	writeQueue                    chan writeRequest
	subscriptionChannelCloseQueue chan *subscription.T

	// custom things that aren't often used
	//
	AssumeValid bool // skip verifying signatures of events from this relay
}

func (r *T) URL() string { return r.url }

func (r *T) Delete(key string) { r.Subscriptions.Delete(key) }

type writeRequest struct {
	msg    []byte
	answer chan error
}

// NewRelay returns a new relay. The relay connection will be closed when the
// context is canceled.
func NewRelay(c context.T, url string, opts ...Option) *T {
	ctx, cancel := context.Cancel(c)
	r := &T{
		url:                           normalize.URL(url),
		ConnectionContext:             ctx,
		ConnectionContextCancel:       cancel,
		Subscriptions:                 xsync.NewMapOf[*subscription.T](),
		okCallbacks:                   xsync.NewMapOf[func(bool, string)](),
		writeQueue:                    make(chan writeRequest),
		subscriptionChannelCloseQueue: make(chan *subscription.T),
		AuthRequired:                  make(chan struct{}),
		Authed:                        make(chan struct{}),
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

// Connect returns a relay object connected to url. Once successfully
// connected, cancelling ctx has no effect. To close the connection, call
// r.Close().
func Connect(c context.T, url string, opts ...Option) (*T, error) {
	r := NewRelay(c, url, opts...)
	err := r.Connect(c)
	return r, err
}

// ConnectWithAuth auths with the relay, checks if its NIP-11 says auth-required
// and uses the provided sec to sign the auth challenge.
func ConnectWithAuth(c context.T, url, sec string,
	opts ...Option) (rl *T, err error) {

	if rl, err = Connect(c, url, opts...); chk.E(err) {
		return
	}
	var inf *relayinfo.T
	if inf, err = relayinfo.Fetch(c, url); chk.E(err) {
		return
	}
	// if NIP-11 doesn't say auth-required, we are done
	if !inf.Limitation.AuthRequired {
		return
	}
	// otherwise, expect auth immediately and sign on it. some relays may not send
	// the auth challenge without being prompted by a req envelope but fuck them.
	// auth-required in nip-11 should mean auth on connect. period.
	authed := false
out:
	for i := 0; i < 2; i++ {
		// but just in case, we will do this twice if need be. The first try may
		// time out because the relay waits for a req, or because the auth
		// doesn't trigger until a message is received.
		select {
		case <-rl.AuthRequired:
			log.T.Ln("authing to relay")
			if err = rl.Auth(c,
				func(evt *event.T) (err error) {
					if err = evt.Sign(sec); chk.E(err) {
						return
					}
					rl.AuthEventID = evt.ID
					return
				}); chk.E(err) {
				return
			}
		case <-time.After(5 * time.Second):
		case <-rl.Authed:
			log.T.Ln("authed to relay", rl.AuthEventID)
			authed = true
		}
		if authed {
			break out
		}
		// to trigger this if auth wasn't immediately demanded, send out a dummy
		// empty req.
		one := 1
		filt := filters.T{
			{Limit: &one},
		}
		var sub *subscription.T
		if sub, err = rl.Subscribe(c, filt); chk.E(err) {
			// not sure what to do here
		}
		sub.Close()
		// at this point if we haven't received an auth there is something wrong
		// with the relay.
	}
	return
}

// When instantiating relay connections, some options may be passed.

// Option is the type of the argument passed for that.
type Option interface {
	IsRelayOption()
}

// WithNoticeHandler just takes notices and is expected to do something with
// them. when not given, defaults to logging the notices.
type WithNoticeHandler func(notice string)

func (_ WithNoticeHandler) IsRelayOption() {}

var _ Option = (WithNoticeHandler)(nil)

// String just returns the relay URL.
func (r *T) String() string {
	return r.url
}

// Context retrieves the context that is associated with this relay connection.
func (r *T) Context() context.T { return r.ConnectionContext }

// IsConnected returns true if the connection to this relay seems to be active.
func (r *T) IsConnected() bool { return r.ConnectionContext.Err() == nil }

// Connect tries to establish a websocket connection to r.URL. If the context
// expires before the connection is complete, an error is returned. Once
// successfully connected, context expiration has no effect: call r.Close to
// close the connection.
//
// The underlying relay connection will use a background context. If you want to
// pass a custom context to the underlying relay connection, use NewRelay() and
// then Relay.Connect().
func (r *T) Connect(c context.T) (err error) {
	if r.ConnectionContext == nil || r.Subscriptions == nil {
		return fmt.Errorf("relay must be initialized with a call to NewRelay()")
	}
	if r.url == "" {
		return fmt.Errorf("invalid relay URL '%s'", r.URL())
	}
	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		c, cancel = context.Timeout(c, 7*time.Second)
		defer cancel()
	}
	var conn *connection.C
	conn, err = connection.NewConnection(c, r.url, r.RequestHeader)
	if err != nil {
		return fmt.Errorf("error opening websocket to '%s': %w", r.URL(), err)
	}
	r.Connection = conn
	// ping every 29 seconds
	ticker := time.NewTicker(29 * time.Second)
	// to be used when the connection is closed
	go func() {
		<-r.ConnectionContext.Done()
		// close these things when the connection is closed
		if r.notices != nil {
			log.I.Ln("closing notices chan")
			close(r.notices)
		}
		// stop the ticker
		ticker.Stop()
		// close all subscriptions
		r.Subscriptions.Range(func(_ string, sub *subscription.T) bool {
			go sub.Unsub()
			return true
		})
	}()

	// queue all write operations here so we don't do mutex spaghetti
	go func() {
		var err error
		for {
			select {
			case <-ticker.C:
				err = wsutil.WriteClientMessage(r.Connection.Conn, ws.OpPing,
					nil)
				if err != nil {
					log.D.F("{%s} error writing ping: %v; closing websocket",
						r.URL(), err)
					chk.D(r.Close()) // this should trigger a context cancelation
					return
				}
			case wr := <-r.writeQueue:
				if wr.msg == nil {
					return
				}
				// all write requests will go through this to prevent races
				if err = r.Connection.WriteMessage(wr.msg); err != nil {
					wr.answer <- err
				}
				close(wr.answer)
			case <-r.ConnectionContext.Done():
				// stop here
				return
			}
		}
	}()

	// general message reader loop
	go r.MessageReadLoop(conn)
	return nil
}

func (r *T) MessageReadLoop(conn *connection.C) {
	buf := new(bytes.Buffer)
	var err error
	for {
		buf.Reset()
		if err = conn.ReadMessage(r.ConnectionContext, buf); err != nil {
			r.ConnectionError = err
			chk.D(r.Close())
			break
		}

		message := buf.Bytes()
		// log.I.F("{%s} received %v", r.URL(), string(message))
		var envelope enveloper.I
		if envelope, _, err = envelopes.ProcessEnvelope(message); chk.E(err) {
			log.I.Ln(string(message))
			continue
		}
		if envelope == nil {
			continue
		}

		switch env := envelope.(type) {
		case *noticeenvelope.T:
			// see WithNoticeHandler
			if r.notices != nil {
				r.notices <- env.Text
			} else {
				log.D.F("NOTICE from %s: '%s'", r.URL(), env.Text)
			}
		case *authenvelope.Challenge:
			r.challenge = env.Challenge
			log.D.Ln("challenge", r.challenge)
			log.I.Ln("signalling auth required")
			r.AuthRequired <- struct{}{}
		case *eventenvelope.T:
			if env.SubscriptionID == "" {
				continue
			}
			if s, ok := r.Subscriptions.Load(env.SubscriptionID.String()); !ok {
				log.D.F("{%s} no subscription with id '%s'",
					r.URL(), env.SubscriptionID.String())
				continue
			} else {
				// check if the event matches the desired filter, ignore otherwise
				if !s.Filters.Match(env.Event) {
					log.D.F("{%s} filter does not match: %v ~ %v",
						r.URL(), s.Filters, env.Event)
					continue
				}
				// check signature, ignore invalid, except from trusted (AssumeValid) relays
				if !r.AssumeValid {
					if ok, err = env.Event.CheckSignature(); !ok {
						errmsg := ""
						if chk.D(err) {
							errmsg = err.Error()
						}
						log.D.F("{%s} bad signature on %s; %s",
							r.URL(), env.Event.ID, errmsg)
						continue
					}
				}
				// dispatch this to the internal .events channel of the
				// subscription
				s.DispatchEvent(env.Event)
			}
		case *eoseenvelope.T:
			log.D.Ln("eose", r.Subscriptions.Size())
			if s, ok := r.Subscriptions.Load(env.Sub.String()); ok {
				log.D.Ln("dispatching eose", env.Sub.String())
				s.DispatchEose()
			}
		case *closedenvelope.T:
			if s, ok := r.Subscriptions.Load(env.ID.String()); ok {
				s.DispatchClosed(env.Reason)
			}
		case *countenvelope.Response:
			if s, ok := r.Subscriptions.Load(env.ID.String()); ok &&
				s.CountResult != nil {
				s.CountResult <- env.Count
			}
		case *okenvelope.T:
			if env.ID == r.AuthEventID {
				close(r.Authed)
			}
			if okCallback, exist := r.okCallbacks.Load(env.ID.String()); exist {
				okCallback(env.OK, env.Reason)
			} else {
				log.D.F("{%s} got an unexpected OK message for event %s",
					r.URL(), env.ID)
			}
		}
	}
}

// Write queues a message to be sent to the relay.
func (r *T) Write(msg []byte) (ch chan error) {
	ch = make(chan error)
	timeout := time.After(time.Second * 5)
	select {
	case r.writeQueue <- writeRequest{msg: msg, answer: ch}:
	case <-r.ConnectionContext.Done():
		ch <- fmt.Errorf("connection closed")
	case <-timeout:
		ch <- fmt.Errorf("write timed out")
		return
	}
	return
}

// Publish sends an "EVENT" command to the relay r as in NIP-01 and waits for an
// OK response.
func (r *T) Publish(c context.T, ev *event.T) error {
	return r.publish(c, ev.ID.String(), &eventenvelope.T{Event: ev})
}

// Auth sends an "AUTH" command client->relay as in NIP-42 and waits for an OK
// response.
func (r *T) Auth(c context.T, sign func(ev *event.T) error) error {
	log.I.Ln("sending auth response to relay", r.URL())
	authEvent := auth.CreateUnsigned(r.challenge, r.URL())
	if err := sign(authEvent); chk.D(err) {
		return fmt.Errorf("error signing auth event: %w", err)
	}
	r.AuthEventID = authEvent.ID
	return r.publish(c, authEvent.ID.String(),
		&authenvelope.Response{Event: authEvent})
}

// publish can be used both for EVENT and for AUTH
func (r *T) publish(c context.T, id string, env enveloper.I) (err error) {
	var cancel context.F
	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 4 seconds
		c, cancel = context.Timeout(c, 4*time.Second)
		defer cancel()
	} else {
		// otherwise make the context cancellable so we can stop everything upon
		// receiving an "OK"
		c, cancel = context.Cancel(c)
		defer cancel()
	}
	// listen for an OK callback
	gotOk := false
	r.okCallbacks.Store(id, func(ok bool, reason string) {
		gotOk = true
		if !ok {
			err = log.E.Err("msg: %s", reason)
		}
		cancel()
	})
	defer r.okCallbacks.Delete(id)
	// publish event
	var enb []byte
	enb, err = env.MarshalJSON()
	// log.T.F("{%s} sending %v", r.URL(), string(enb))
	if err = <-r.Write(enb); err != nil {
		return err
	}
	for {
		select {
		case <-c.Done():
			// this will be called when we get an OK or when the context has been canceled
			if gotOk {
				return err
			}
			return c.Err()
		case <-r.ConnectionContext.Done():
			// this is caused when we lose connectivity
			return err
		}
	}
}

// Subscribe sends a "REQ" command to the relay r as in NIP-01. Events are
// returned through the channel sub.Events. The subscription is closed when
// context ctx is cancelled ("CLOSE" in NIP-01).
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or
// ensuring their `context.T` will be canceled at some point. Failure to do that
// will result in a huge number of halted goroutines being created.
func (r *T) Subscribe(c context.T, f filters.T,
	opts ...subscriptionoption.I) (*subscription.T, error) {

	sub := r.PrepareSubscription(c, f, opts...)

	if err := sub.Fire(); err != nil {
		return nil, fmt.Errorf("couldn't subscribe to %v at %s: %w", f, r.URL(),
			err)
	}

	return sub, nil
}

// PrepareSubscription creates a subscription, but doesn't fire it.
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or
// ensuring their `context.T` will be canceled at some point. Failure to do that
// will result in a huge number of halted goroutines being created.
func (r *T) PrepareSubscription(c context.T, f filters.T,
	opts ...subscriptionoption.I) *subscription.T {

	if r.Connection == nil {
		panic(fmt.Errorf("must call .Connect() first before calling .Subscribe()"))
	}

	current := subscriptionIDCounter.Add(1)
	ctx, cancel := context.Cancel(c)

	sub := &subscription.T{
		Relay:             r,
		Context:           ctx,
		Cancel:            cancel,
		Counter:           int(current),
		Events:            make(event.C),
		EndOfStoredEvents: make(chan struct{}),
		ClosedReason:      make(chan string, 1),
		Filters:           f,
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case subscription.WithLabel:
			sub.Label = string(o)
		}
	}

	id := sub.GetID()
	r.Subscriptions.Store(id, sub)

	// start handling events, eose, unsub etc:
	go sub.Start()

	return sub
}

func (r *T) QuerySync(c context.T, f *filter.T,
	opts ...subscriptionoption.I) ([]*event.T, error) {
	log.D.Ln(f.ToObject().String())
	sub, err := r.Subscribe(c, filters.T{f}, opts...)
	if err != nil {
		return nil, err
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
				log.I.Ln("channel is closed")
				return events, nil
			}
			events = append(events, evt)
		case <-sub.EndOfStoredEvents:
			log.I.Ln("EOSE")
			return events, nil
		case <-c.Done():
			log.I.Ln("sub context done")
			return events, nil
		}
	}
}

func (r *T) Count(c context.T, filters filters.T,
	opts ...subscriptionoption.I) (int, error) {

	sub := r.PrepareSubscription(c, filters, opts...)
	sub.CountResult = make(chan int)

	if err := sub.Fire(); err != nil {
		return 0, err
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
		case count := <-sub.CountResult:
			return count, nil
		case <-c.Done():
			return 0, c.Err()
		}
	}
}

func (r *T) Close() error {
	r.closeMutex.Lock()
	defer r.closeMutex.Unlock()

	if r.ConnectionContextCancel == nil {
		return fmt.Errorf("relay not connected")
	}

	r.ConnectionContextCancel()
	r.ConnectionContextCancel = nil
	return r.Connection.Conn.Close()
}
