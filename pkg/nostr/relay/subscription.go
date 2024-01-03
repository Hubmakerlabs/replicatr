package relay

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	close2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/close"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip45"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
)

type Subscription struct {
	label   string
	counter int
	Relay   *Relay
	Filters filters.T
	// for this to be treated as a COUNT and not a REQ this must be set
	CountResult chan int64
	// the Events channel emits all EVENTs that come in a Subscription
	// will be closed when the subscription ends
	Events chan *event.T
	mu     sync.Mutex
	// the EndOfStoredEvents channel gets closed when an EOSE comes for that subscription
	EndOfStoredEvents chan struct{}
	// Context will be .Done() when the subscription ends
	Context context.Context
	Live    atomic.Bool
	Eosed   atomic.Bool
	Cancel  context.CancelFunc
	// this keeps track of the events we've received before the EOSE that we must dispatch before
	// closing the EndOfStoredEvents channel
	Storedwg sync.WaitGroup
}

type EventMessage struct {
	Event event.T
	Relay string
}

// When instantiating relay connections, some options may be passed.

// SubscriptionOption is the type of the argument passed for that.
// Some examples are WithLabel.
type SubscriptionOption interface {
	IsSubscriptionOption()
}

// WithLabel puts a label on the subscription (it is prepended to the automatic id) that is sent to relays.
type WithLabel string

func (_ WithLabel) IsSubscriptionOption() {}

var _ SubscriptionOption = (WithLabel)("")

// GetID return the Nostr subscription ID as given to the Relay it is a
// concatenation of the label and a serial number.
func (sub *Subscription) GetID() string {
	return sub.label + ":" + strconv.Itoa(sub.counter)
}

func (sub *Subscription) Start() {
	<-sub.Context.Done()
	// the subscription ends once the context is canceled (if not already)
	sub.Unsub() // this will set sub.Live to false
	// do this so we don't have the possibility of closing the Events channel
	// and then trying to send to it
	sub.mu.Lock()
	close(sub.Events)
	sub.mu.Unlock()
}

func (sub *Subscription) DispatchEvent(evt *event.T) {
	added := false
	if !sub.Eosed.Load() {
		sub.Storedwg.Add(1)
		added = true
	}
	go func() {
		sub.mu.Lock()
		defer sub.mu.Unlock()
		if sub.Live.Load() {
			select {
			case sub.Events <- evt:
			case <-sub.Context.Done():
			}
		}
		if added {
			sub.Storedwg.Done()
		}
	}()
}

func (sub *Subscription) dispatchEose() {
	if sub.Eosed.CompareAndSwap(false, true) {
		go func() {
			sub.Storedwg.Wait()
			close(sub.EndOfStoredEvents)
		}()
	}
}

// Unsub closes the subscription, sending "CLOSE" to relay as in NIP-01.
// Unsub() also closes the channel sub.Events and makes a new one.
func (sub *Subscription) Unsub() {
	// Cancel the context (if it's not canceled already)
	sub.Cancel()
	// mark subscription as closed and send a CLOSE to the relay (naïve sync.Once implementation)
	if sub.Live.CompareAndSwap(true, false) {
		sub.Close()
	}
	// remove subscription from our map
	sub.Relay.Subscriptions.Delete(sub.GetID())
}

// Close just sends a CLOSE message. You probably want Unsub() instead.
func (sub *Subscription) Close() {
	if sub.Relay.IsConnected() {
		id := sub.GetID()
		closeMsg := &close2.Envelope{T: subscriptionid.T(id)}
		closeb, e := closeMsg.MarshalJSON()
		log.D.Chk(e)
		log.D.F("{%s} sending %s", sub.Relay.URL, string(closeb))
		<-sub.Relay.Write(closeb)
	}
}

// Sub sets sub.T and then calls sub.Fire(ctx).
// The subscription will be closed if the context expires.
func (sub *Subscription) Sub(_ context.Context, filters filters.T) {
	sub.Filters = filters
	if e := sub.Fire(); fails(e) {
	}
}

// Fire sends the "REQ" command to the relay.
func (sub *Subscription) Fire() (e error) {
	id := sub.GetID()
	var reqb []byte
	if sub.CountResult == nil {
		if reqb, e = (&nip1.ReqEnvelope{
			SubscriptionID: subscriptionid.T(id),
			T:              sub.Filters,
		}).MarshalJSON(); fails(e) {
		}
	} else {
		if reqb, e = (&nip45.CountRequestEnvelope{
			SubscriptionID: subscriptionid.T(id),
			T:              sub.Filters,
		}).MarshalJSON(); fails(e) {
		}
	}
	log.D.F("{%s} sending %v", sub.Relay.URL, string(reqb))
	sub.Live.Store(true)
	if e = <-sub.Relay.Write(reqb); fails(e) {
		sub.Cancel()
		return fmt.Errorf("failed to write: %w", e)
	}

	return nil
}
