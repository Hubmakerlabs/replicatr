package nostr

import (
	"context"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip45"
	"strconv"
	"sync"
	"sync/atomic"
)

type Subscription struct {
	label   string
	counter int
	Relay   *Relay
	Filters nip1.Filters
	// for this to be treated as a COUNT and not a REQ this must be set
	countResult chan int64
	// the Events channel emits all EVENTs that come in a Subscription
	// will be closed when the subscription ends
	Events chan *nip1.Event
	mu     sync.Mutex
	// the EndOfStoredEvents channel gets closed when an EOSE comes for that subscription
	EndOfStoredEvents chan struct{}
	// Context will be .Done() when the subscription ends
	Context context.Context
	live    atomic.Bool
	eosed   atomic.Bool
	cancel  context.CancelFunc
	// this keeps track of the events we've received before the EOSE that we must dispatch before
	// closing the EndOfStoredEvents channel
	storedwg sync.WaitGroup
}

type EventMessage struct {
	Event nip1.Event
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

func (sub *Subscription) start() {
	<-sub.Context.Done()
	// the subscription ends once the context is canceled (if not already)
	sub.Unsub() // this will set sub.live to false
	// do this so we don't have the possibility of closing the Events channel
	// and then trying to send to it
	sub.mu.Lock()
	close(sub.Events)
	sub.mu.Unlock()
}

func (sub *Subscription) dispatchEvent(evt *nip1.Event) {
	added := false
	if !sub.eosed.Load() {
		sub.storedwg.Add(1)
		added = true
	}
	go func() {
		sub.mu.Lock()
		defer sub.mu.Unlock()
		if sub.live.Load() {
			select {
			case sub.Events <- evt:
			case <-sub.Context.Done():
			}
		}
		if added {
			sub.storedwg.Done()
		}
	}()
}

func (sub *Subscription) dispatchEose() {
	if sub.eosed.CompareAndSwap(false, true) {
		go func() {
			sub.storedwg.Wait()
			close(sub.EndOfStoredEvents)
		}()
	}
}

// Unsub closes the subscription, sending "CLOSE" to relay as in NIP-01.
// Unsub() also closes the channel sub.Events and makes a new one.
func (sub *Subscription) Unsub() {
	// cancel the context (if it's not canceled already)
	sub.cancel()
	// mark subscription as closed and send a CLOSE to the relay (naïve sync.Once implementation)
	if sub.live.CompareAndSwap(true, false) {
		sub.Close()
	}
	// remove subscription from our map
	sub.Relay.Subscriptions.Delete(sub.GetID())
}

// Close just sends a CLOSE message. You probably want Unsub() instead.
func (sub *Subscription) Close() {
	if sub.Relay.IsConnected() {
		id := sub.GetID()
		closeMsg := &nip1.CloseEnvelope{SubscriptionID: nip1.SubscriptionID(id)}
		closeb, e := closeMsg.MarshalJSON()
		log.D.Chk(e)
		log.D.F("{%s} sending %s", sub.Relay.URL, closeb)
		<-sub.Relay.Write(closeb)
	}
}

// Sub sets sub.Filters and then calls sub.Fire(ctx).
// The subscription will be closed if the context expires.
func (sub *Subscription) Sub(_ context.Context, filters nip1.Filters) {
	sub.Filters = filters
	if e := sub.Fire(); fails(e) {
	}
}

// Fire sends the "REQ" command to the relay.
func (sub *Subscription) Fire() (e error) {
	id := sub.GetID()
	var reqb []byte
	if sub.countResult == nil {
		if reqb, e = (&nip1.ReqEnvelope{
			SubscriptionID: nip1.SubscriptionID(id),
			Filters:        sub.Filters,
		}).MarshalJSON(); fails(e) {
		}
	} else {
		if reqb, e = (&nip45.CountRequestEnvelope{
			SubscriptionID: nip1.SubscriptionID(id),
			Filters:        sub.Filters,
		}).MarshalJSON(); fails(e) {
		}
	}
	log.D.F("{%s} sending %v", sub.Relay.URL, string(reqb))
	sub.live.Store(true)
	if e = <-sub.Relay.Write(reqb); fails(e) {
		sub.cancel()
		return fmt.Errorf("failed to write: %w", e)
	}

	return nil
}
