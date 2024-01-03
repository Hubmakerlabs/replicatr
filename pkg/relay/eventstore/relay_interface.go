package eventstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
)

// RelayInterface is a wrapper thing that unifies Store and nostr.Relay under a common API.
type RelayInterface interface {
	Publish(ctx context.Context, event nip1.Event) error
	QuerySync(ctx context.Context, filter *nip1.Filter, opts ...SubscriptionOption) ([]*nip1.Event, error)
}

// SubscriptionOption is the type of the argument passed for that.
// Some examples are WithLabel.
type SubscriptionOption interface {
	IsSubscriptionOption()
}

// WithLabel puts a label on the subscription (it is prepended to the automatic id) that is sent to relays.
type WithLabel string

func (_ WithLabel) IsSubscriptionOption() {}

// compile time interface check
var _ SubscriptionOption = (WithLabel)("")

type RelayWrapper struct {
	Store
}

// compile time interface check
var _ RelayInterface = (*RelayWrapper)(nil)

func (w RelayWrapper) Publish(ctx context.Context, evt nip1.Event) (e error) {
	if evt.Kind.IsEphemeral() {
		// do not store ephemeral events
		return nil
	} else if evt.Kind.IsReplaceable() {
		// replaceable event, delete before storing
		var ch chan *nip1.Event
		ch, e = w.Store.QueryEvents(ctx, &nip1.Filter{Authors: []string{evt.PubKey}, Kinds: kinds.T{evt.Kind}})
		if fails(e) {
			return fmt.Errorf("failed to query before replacing: %w", e)
		}
		if previous := <-ch; previous != nil && isOlder(previous, &evt) {
			if e = w.Store.DeleteEvent(ctx, previous); fails(e) {
				return fmt.Errorf("failed to delete event for replacing: %w", e)
			}
		}
	} else if evt.Kind.IsParameterizedReplaceable() {
		// parameterized replaceable event, delete before storing
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d != nil {
			var ch chan *nip1.Event
			ch, e = w.Store.QueryEvents(ctx, &nip1.Filter{
				Authors: []string{evt.PubKey},
				Kinds:   kinds.T{evt.Kind},
				Tags:    nip1.TagMap{"d": []string{d.Value()}},
			})
			if fails(e) {
				return fmt.Errorf(
					"failed to query before parameterized replacing: %w", e)
			}
			if previous := <-ch; previous != nil && isOlder(previous, &evt) {
				if e = w.Store.DeleteEvent(ctx, previous); fails(e) {
					return fmt.Errorf(
						"failed to delete event for parameterized replacing: %w", e)
				}
			}
		}
	}
	if e = w.SaveEvent(ctx, &evt); fails(e) && !errors.Is(e, ErrDupEvent) {
		return fmt.Errorf("failed to save: %w", e)
	}
	return nil
}

func (w RelayWrapper) QuerySync(ctx context.Context, filter *nip1.Filter,
	opts ...SubscriptionOption) (evs []*nip1.Event,e error) {
	var ch chan *nip1.Event
	if ch, e= w.Store.QueryEvents(ctx, filter); log.E.Chk(e){
		return nil, fmt.Errorf("failed to query: %w", e)
	}

	n := filter.Limit
	if n == 0 {
		n = 500
	}

	results := make([]*nip1.Event, 0, n)
	for evt := range ch {
		results = append(results, evt)
	}

	return results, nil
}
