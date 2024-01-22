package replicatr

import (
	"errors"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

// AddEvent sends an event through then normal add pipeline, as if it was
// received from a websocket.
func (rl *Relay) AddEvent(c context.T, ev *event.T) (e error) {
	if ev == nil {
		e = errors.New("error: event is nil")
		rl.E.Ln(e)
		return
	}
	for _, rej := range rl.RejectEvent {
		if reject, msg := rej(c, ev); reject {
			if msg == "" {
				e = errors.New("blocked: no reason")
				rl.E.Ln(e)
				return
			} else {
				e = errors.New(normalize.OKMessage(msg, "blocked"))
				rl.E.Ln(e)
				return
			}
		}
	}
	if 20000 <= ev.Kind && ev.Kind < 30000 {
		// do not store ephemeral events
	} else {
		if ev.Kind == 0 || ev.Kind == 3 || (10000 <= ev.Kind && ev.Kind < 20000) {
			// replaceable event, delete before storing
			for _, query := range rl.QueryEvents {
				var ch chan *event.T
				ch, e = query(c, &filter.T{
					Authors: tag.T{ev.PubKey},
					Kinds:   kinds.T{ev.Kind},
				})
				if rl.E.Chk(e) {
					continue
				}
				if previous := <-ch; previous != nil && isOlder(previous, ev) {
					for _, del := range rl.DeleteEvent {
						rl.D.Chk(del(c, previous))
					}
				}
			}
		} else if 30000 <= ev.Kind && ev.Kind < 40000 {
			// parameterized replaceable event, delete before storing
			d := ev.Tags.GetFirst([]string{"d", ""})
			if d != nil {
				for _, query := range rl.QueryEvents {
					var ch chan *event.T
					if ch, e = query(c, &filter.T{
						Authors: tag.T{ev.PubKey},
						Kinds:   kinds.T{ev.Kind},
						Tags:    filter.TagMap{"d": []string{d.Value()}},
					}); rl.E.Chk(e) {
						continue
					}
					if previous := <-ch; previous != nil && isOlder(previous, ev) {
						for _, del := range rl.DeleteEvent {
							rl.E.Chk(del(c, previous))
						}
					}
				}
			}
		}
		// store
		for _, store := range rl.StoreEvent {
			if saveErr := store(c, ev); rl.E.Chk(saveErr) {
				switch {
				case errors.Is(saveErr, eventstore.ErrDupEvent):
					return nil
				default:
					return fmt.Errorf(normalize.OKMessage(saveErr.Error(), "error"))
				}
			}
		}
		for _, ons := range rl.OnEventSaved {
			ons(c, ev)
		}
	}
	for _, ovw := range rl.OverwriteResponseEvent {
		ovw(c, ev)
	}
	rl.BroadcastEvent(ev)
	return nil
}
