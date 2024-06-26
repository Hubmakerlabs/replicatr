package app

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/okenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

// handleDeleteRequest handles a delete event (kind 5)
func (rl *Relay) handleDeleteRequest(c context.T, evt *event.T) (err error) {
	// log.I.Ln("event delete", evt.ToObject().String())
	// event deletion -- nip-09
	go func() {
		for _, t := range evt.Tags {
			if len(t) >= 2 && t[0] == "e" {
				// log.I.Ln("delete event tag", t)
				// first we fetch the event
				for _, query := range rl.QueryEvents {
					var ch chan *event.T
					if ch, err = query(c, &filter.T{IDs: tag.T{t[1]}}); chk.E(err) {
						continue
					}
					// log.I.Ln("waiting for result")
					target := <-ch
					if target == nil {
						continue
					}
					// log.I.Ln("got result", ok, target.ToObject().String())
					// got the event, now check if the user can delete it
					acceptDeletion := target.PubKey == evt.PubKey
					// todo: enable administrators to do this
					var msg string
					if acceptDeletion == false {
						msg = "you are not the author of this event"
					}
					// but if we have a function to override this outcome, use that
					// instead
					for _, odo := range rl.OverrideDeletion {
						var override bool
						override, msg = odo(c, target, evt)
						// if any override rejects, do not delete
						acceptDeletion = acceptDeletion && override
					}
					// at this point only if the pubkey matches AND the overrides
					// all accept will the deletion be performed.
					if acceptDeletion {
						// delete it
						for _, del := range rl.DeleteEvent {
							chk.E(del(c, target))
						}
					} else {
						// fail and stop here
						err = fmt.Errorf("blocked: %s", msg)
						// log.E.Ln(err)
						return
					}
					// don't try to query this same event again
					break
				}
			}
		}
	}()
	return nil
}

// OverrideDelete decides whether to veto a delete event.
//
// Temporarily removing delete functionality until a proper tombstone/indexing
// strategy is devised to filter out these events from database results.
func (rl *Relay) OverrideDelete(c context.T, tgt, del *event.T) (ok bool,
	msg string) {
	log.T.Ln("overriding delete")
	msg = normalize.Reason("not actually deleting", okenvelope.Blocked.S())
	ok = false
	return
}
