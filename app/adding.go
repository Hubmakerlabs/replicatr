package app

import (
	"errors"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
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
	// var ch chan *event.T
	// defer close(ch)
	if ev.Kind.IsEphemeral() {
		log.D.Ln("ephemeral event")
		// do not store ephemeral events
	} else {
		// todo: this seems to be unnecessary for badger
		// if ev.Kind.IsReplaceable() {
		// 	log.D.Ln("replaceable event")
		// 	// replaceable event, delete before storing
		// 	for i, query := range rl.QueryEvents {
		// 		log.D.Ln("running query", i)
		// 		ch, err = query(c, &filter.T{
		// 			Authors: tag.T{ev.PubKey},
		// 			Kinds:   kinds.T{ev.Kind},
		// 		})
		// 		if chk.E(err) {
		// 			continue
		// 		}
		// 		if previous := <-ch; previous != nil && isOlder(previous, ev) {
		// 			for _, del := range rl.DeleteEvent {
		// 				log.D.Chk(del(c, previous))
		// 			}
		// 		}
		// 	}
		// 	log.D.Ln("finished replaceable event")
		// } else if ev.Kind.IsParameterizedReplaceable() {
		// 	log.D.Ln("parameterized replaceable event")
		// 	// parameterized replaceable event, delete before storing
		// 	d := ev.Tags.GetFirst([]string{"d", ""})
		// 	if d != nil {
		// 		for _, query := range rl.QueryEvents {
		// 			if ch, err = query(c, &filter.T{
		// 				Authors: tag.T{ev.PubKey},
		// 				Kinds:   kinds.T{ev.Kind},
		// 				Tags:    filter.TagMap{"d": []string{d.Value()}},
		// 			}); chk.E(err) {
		// 				continue
		// 			}
		// 			if previous := <-ch; previous != nil && isOlder(previous, ev) {
		// 				for _, del := range rl.DeleteEvent {
		// 					chk.E(del(c, previous))
		// 				}
		// 			}
		// 		}
		// 	}
		// }
		log.D.Ln("storing event")
		// store
		for i, store := range rl.StoreEvent {
			log.D.Ln("running event store function", i, ev.ToObject().String())
			if saveErr := store(c, ev); chk.T(saveErr) {
				switch {
				case errors.Is(saveErr, eventstore.ErrDupEvent):
					log.D.Ln(saveErr)
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
	}
	for _, ovw := range rl.OverwriteResponseEvent {
		ovw(c, ev)
	}
	rl.BroadcastEvent(ev)
	return nil
}
