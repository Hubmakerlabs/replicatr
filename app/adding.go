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
		// log.I.Ln("adding event")
		for _, store := range rl.StoreEvent {
			if saveErr := store(c, ev); chk.E(saveErr) {
				switch {
				case errors.Is(saveErr, eventstore.ErrDupEvent):
					return saveErr
				default:
					err = log.E.Err(normalize.Reason(saveErr.Error(), "error"))
					log.D.Ln(ev.ID, err)
					return
				}
			}
		}
		for _, ons := range rl.OnEventSaved {
			ons(c, ev)
		}
	} else {
		log.I.Ln("ephemeral event")
	}
	for _, ovw := range rl.OverwriteResponseEvent {
		ovw(c, ev)
	}
	rl.BroadcastEvent(ev)
	return nil
}
