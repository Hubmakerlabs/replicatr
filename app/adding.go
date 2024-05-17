package app

import (
	"errors"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
)

// AddEvent sends an event through then normal add pipeline, as if it was
// received from a websocket.
func (rl *Relay) AddEvent(c context.T, ev *event.T) (err error) {
	if !rl.IsAuthed(c, "add event") {
		return
	}
	if ev == nil {
		err = errors.New("error: event is nil")
		log.E.Ln(err)
		return
	}
	for _, rej := range rl.RejectEvent {
		if reject, msg := rej(c, ev); reject {
			if msg == "" {
				err = errors.New("blocked: no reason")
				log.E.Ln(err)
				return
			} else {
				err = errors.New(normalize.Reason(msg, "blocked"))
				log.E.Ln(err)
				return
			}
		}
	}
	if !ev.Kind.IsEphemeral() {
		// log.I.Ln("adding event", ev.ToObject().String())
		for _, store := range rl.StoreEvent {
			if saveErr := store(c, ev); saveErr != nil {
				switch {
				case errors.Is(saveErr, eventstore.ErrDupEvent):
					return saveErr
				default:
					err = log.E.Err(normalize.Reason(saveErr.Error(), "error"))
					log.D.Ln(ev.ID, err)
					return
				}
			}
			// log.I.Ln("added event", ev.ID, "for", GetAuthed(c))
		}
		for _, ons := range rl.OnEventSaved {
			ons(c, ev)
		}
		// log.I.Ln("saved event", ev.ID)
	} else {
		// log.I.Ln("ephemeral event")
		return
	}
	for _, ovw := range rl.OverwriteResponseEvent {
		ovw(c, ev)
	}
	rl.BroadcastEvent(ev)
	return nil
}
