package replicatr

import (
	"context"
	"errors"
	"fmt"

	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
)

// AddEvent sends an event through then normal add pipeline, as if it was received from a websocket.
func (rl *Relay) AddEvent(ctx context.Context, evt *nostr.Event) (e error) {
	if evt == nil {
		return errors.New("error: event is nil")
	}
	for _, rejectors := range rl.RejectEvent {
		if reject, msg := rejectors(ctx, evt); reject {
			if msg == "" {
				e = errors.New("blocked: no reason")
				rl.Log.E.Ln(e)
				return
			} else {
				e = errors.New(nostr.NormalizeOKMessage(msg, "blocked"))
				rl.Log.E.Ln(e)
				return
			}
		}
	}
	if 20000 <= evt.Kind && evt.Kind < 30000 {
		// do not store ephemeral events
	} else {
		if evt.Kind == 0 || evt.Kind == 3 || (10000 <= evt.Kind && evt.Kind < 20000) {
			// replaceable event, delete before storing
			for _, query := range rl.QueryEvents {
				var ch chan *nostr.Event
				ch, e = query(ctx, &nostr.Filter{Authors: []string{evt.PubKey}, Kinds: []int{evt.Kind}})
				if rl.Log.E.Chk(e) {
					continue
				}
				if previous := <-ch; previous != nil && isOlder(previous, evt) {
					for _, del := range rl.DeleteEvent {
						rl.Log.D.Chk(del(ctx, previous))
					}
				}
			}
		} else if 30000 <= evt.Kind && evt.Kind < 40000 {
			// parameterized replaceable event, delete before storing
			d := evt.Tags.GetFirst([]string{"d", ""})
			if d != nil {
				for _, query := range rl.QueryEvents {
					var ch chan *nostr.Event
					if ch, e = query(ctx, &nostr.Filter{
						Authors: []string{evt.PubKey},
						Kinds:   []int{evt.Kind},
						Tags:    nostr.TagMap{"d": []string{d.Value()}},
					}); rl.Log.E.Chk(e) {
						continue
					}
					if previous := <-ch; previous != nil && isOlder(previous, evt) {
						for _, del := range rl.DeleteEvent {
							rl.Log.E.Chk(del(ctx, previous))
						}
					}
				}
			}
		}
		// store
		for _, store := range rl.StoreEvent {
			if saveErr := store(ctx, evt); rl.Log.E.Chk(saveErr){
				switch {
				case errors.Is(saveErr, eventstore.ErrDupEvent):
					return nil
				default:
					return fmt.Errorf(nostr.NormalizeOKMessage(saveErr.Error(), "error"))
				}
			}
		}
		for _, ons := range rl.OnEventSaved {
			ons(ctx, evt)
		}
	}
	for _, ovw := range rl.OverwriteResponseEvent {
		ovw(ctx, evt)
	}
	notifyListeners(evt)
	return nil
}
