package eventstore

import (
	"errors"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/interfaces/subscriptionoption"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
)

// RelayInterface is a wrapper thing that unifies Store and nostr.Relay under a common API.
type RelayInterface interface {
	Publish(c context.T, evt *event.T) error
	QuerySync(c context.T, f *filter.T,
		opts ...subscriptionoption.I) ([]*event.T, error)
}

type RelayWrapper struct {
	Store
}

var _ RelayInterface = (*RelayWrapper)(nil)

func (w RelayWrapper) Publish(c context.T, evt *event.T) (e error) {
	var ch chan *event.T
	if 20000 <= evt.Kind && evt.Kind < 30000 {
		// do not store ephemeral events
		return nil
	} else if evt.Kind == 0 || evt.Kind == 3 || (10000 <= evt.Kind && evt.Kind < 20000) {
		// replaceable event, delete before storing
		ch, e = w.Store.QueryEvents(c, &filter.T{
			Authors: []string{evt.PubKey},
			Kinds:   kinds.T{evt.Kind},
		})
		if e != nil {
			return fmt.Errorf("failed to query before replacing: %w", e)
		}
		if previous := <-ch; previous != nil && isOlder(previous, evt) {
			if e := w.Store.DeleteEvent(c, previous); e != nil {
				return fmt.Errorf("failed to delete event for replacing: %w", e)
			}
		}
	} else if 30000 <= evt.Kind && evt.Kind < 40000 {
		// parameterized replaceable event, delete before storing
		d := evt.Tags.GetFirst([]string{"d", ""})
		if d != nil {
			ch, e = w.Store.QueryEvents(c, &filter.T{
				Authors: []string{evt.PubKey},
				Kinds:   kinds.T{evt.Kind},
				Tags:    filter.TagMap{"d": []string{d.Value()}},
			})
			if e != nil {
				return fmt.Errorf("failed to query before parameterized replacing: %w", e)
			}
			if previous := <-ch; previous != nil && isOlder(previous, evt) {
				if e = w.Store.DeleteEvent(c, previous); log.Fail(e) {
					return fmt.Errorf("failed to delete event for parameterized replacing: %w", e)
				}
			}
		}
	}
	if e = w.SaveEvent(c, evt); e != nil && !errors.Is(e, ErrDupEvent) {
		return fmt.Errorf("failed to save: %w", e)
	}
	return nil
}

func (w RelayWrapper) QuerySync(c context.T, f *filter.T,
	opts ...subscriptionoption.I) ([]*event.T, error) {

	ch, e := w.Store.QueryEvents(c, f)
	if e != nil {
		return nil, fmt.Errorf("failed to query: %w", e)
	}
	n := f.Limit
	if n == 0 {
		n = 500
	}
	results := make([]*event.T, 0, n)
	for evt := range ch {
		results = append(results, evt)
	}

	return results, nil
}
