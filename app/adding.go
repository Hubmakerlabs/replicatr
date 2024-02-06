package app

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
func (rl *Relay) AddEvent(c context.T, ev *event.T) (err error) {
	if ev == nil {
		err = errors.New("error: event is nil")
		rl.E.Ln(err)
		return
	}
	for _, rej := range rl.RejectEvent {
		if reject, msg := rej(c, ev); reject {
			if msg == "" {
				err = errors.New("blocked: no reason")
				rl.E.Ln(err)
				return
			} else {
				err = errors.New(normalize.OKMessage(msg, "blocked"))
				rl.E.Ln(err)
				return
			}
		}
	}
	if ev.Kind.IsEphemeral() {
		rl.T.Ln("ephemeral event")
		// do not store ephemeral events
	} else {
		if ev.Kind.IsReplaceable() {
			rl.T.Ln("replaceable event")
			// replaceable event, delete before storing
			for _, query := range rl.QueryEvents {
				var ch chan *event.T
				ch, err = query(c, &filter.T{
					Authors: tag.T{ev.PubKey},
					Kinds:   kinds.T{ev.Kind},
				})
				if rl.E.Chk(err) {
					continue
				}
				if previous := <-ch; previous != nil && isOlder(previous, ev) {
					for _, del := range rl.DeleteEvent {
						rl.D.Chk(del(c, previous))
					}
				}
			}
		} else if ev.Kind.IsParameterizedReplaceable() {
			rl.T.Ln("parameterized replaceable event")
			// parameterized replaceable event, delete before storing
			d := ev.Tags.GetFirst([]string{"d", ""})
			if d != nil {
				for _, query := range rl.QueryEvents {
					var ch chan *event.T
					if ch, err = query(c, &filter.T{
						Authors: tag.T{ev.PubKey},
						Kinds:   kinds.T{ev.Kind},
						Tags:    filter.TagMap{"d": []string{d.Value()}},
					}); rl.E.Chk(err) {
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
		for i, store := range rl.StoreEvent {
			rl.T.Ln("running event store function", i)
			if saveErr := store(c, ev); rl.T.Chk(saveErr) {
				switch {
				case errors.Is(saveErr, eventstore.ErrDupEvent):
					rl.T.Ln(saveErr)
					return saveErr
				default:
					err = fmt.Errorf(normalize.OKMessage(saveErr.Error(), "error"))
					rl.T.Ln(ev.ID, err)
					return
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
