package replicatr

import (
	err "errors"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

// AddEvent sends an event through then normal add pipeline, as if it was
// received from a websocket.
func (rl *Relay) AddEvent(ctx Ctx, evt *Event) (e error) {
	if evt == nil {
		e = err.New("error: event is nil")
		rl.E.Ln(e)
		return
	}
	for _, rej := range rl.RejectEvent {
		if reject, msg := rej(ctx, evt); reject {
			if msg == "" {
				e = err.New("blocked: no reason")
				rl.E.Ln(e)
				return
			} else {
				e = err.New(normalize.OKMessage(msg, "blocked"))
				rl.E.Ln(e)
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
				var ch chan *Event
				ch, e = query(ctx, &Filter{
					Authors: tag.T{evt.PubKey},
					Kinds:   []int{evt.Kind},
				})
				if rl.E.Chk(e) {
					continue
				}
				if previous := <-ch; previous != nil && isOlder(previous, evt) {
					for _, del := range rl.DeleteEvent {
						rl.D.Chk(del(ctx, previous))
					}
				}
			}
		} else if 30000 <= evt.Kind && evt.Kind < 40000 {
			// parameterized replaceable event, delete before storing
			d := evt.Tags.GetFirst([]string{"d", ""})
			if d != nil {
				for _, query := range rl.QueryEvents {
					var ch chan *Event
					if ch, e = query(ctx, &Filter{
						Authors: tag.T{evt.PubKey},
						Kinds:   []int{evt.Kind},
						Tags:    TagMap{"d": []string{d.Value()}},
					}); rl.E.Chk(e) {
						continue
					}
					if previous := <-ch; previous != nil && isOlder(previous, evt) {
						for _, del := range rl.DeleteEvent {
							rl.E.Chk(del(ctx, previous))
						}
					}
				}
			}
		}
		// store
		for _, store := range rl.StoreEvent {
			if saveErr := store(ctx, evt); rl.E.Chk(saveErr) {
				switch {
				case err.Is(saveErr, eventstore.ErrDupEvent):
					return nil
				default:
					return fmt.Errorf(normalize.OKMessage(saveErr.Error(), "error"))
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
	rl.BroadcastEvent(evt)
	return nil
}
