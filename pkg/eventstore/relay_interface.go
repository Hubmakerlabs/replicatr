package eventstore

import (
	"context"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
)

// RelayInterface is a wrapper thing that unifies Store and nostr.Relay under a common API.
type RelayInterface interface {
	Publish(ctx context.Context, event nip1.Event) error
	QuerySync(ctx context.Context, filter nip1.Filter) ([]*nip1.Event, error)
	// , opts ...nostr.SubscriptionOption)
}

type RelayWrapper struct {
	Store
}

var _ RelayInterface = (*RelayWrapper)(nil)

func (w RelayWrapper) Publish(ctx context.Context, evt nip1.Event) error {
	if 20000 <= evt.Kind && evt.Kind < 30000 {
		// do not store ephemeral events
		return nil
	} else if evt.Kind == 0 || evt.Kind == 3 || (10000 <= evt.Kind && evt.Kind < 20000) {
		// replaceable event, delete before storing
		ch, err := w.Store.QueryEvents(ctx,
			nip1.Filter{Authors: []string{evt.PubKey},
				Kinds: kind.Array{evt.Kind}})
		if err != nil {
			return fmt.Errorf("failed to query before replacing: %w", err)
		}
		if previous := <-ch; previous != nil && isOlder(previous, &evt) {
			if err := w.Store.DeleteEvent(ctx, previous); err != nil {
				return fmt.Errorf("failed to delete event for replacing: %w",
					err)
			}
		}
	} else if 30000 <= evt.Kind && evt.Kind < 40000 {
		// parameterized replaceable event, delete before storing
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d != nil {
			ch, err := w.Store.QueryEvents(ctx,
				nip1.Filter{Authors: []string{evt.PubKey},
					Kinds: kind.Array{evt.Kind},
					Tags:  nip1.TagMap{"d": []string{d.Value()}}})
			if err != nil {
				return fmt.Errorf("failed to query before parameterized replacing: %w",
					err)
			}
			if previous := <-ch; previous != nil && isOlder(previous, &evt) {
				if err := w.Store.DeleteEvent(ctx, previous); err != nil {
					return fmt.Errorf("failed to delete event for parameterized replacing: %w",
						err)
				}
			}
		}
	}

	if err := w.SaveEvent(ctx, &evt); err != nil && err != ErrDupEvent {
		return fmt.Errorf("failed to save: %w", err)
	}

	return nil
}

func (w RelayWrapper) QuerySync(ctx context.Context, filter nip1.Filter,

// opts ...nostr.SubscriptionOption
) ([]*nip1.Event, error) {
	ch, err := w.Store.QueryEvents(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
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
