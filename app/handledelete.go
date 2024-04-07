package app

import (
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/tag"
)

// handleDeleteRequest handles a delete event (kind 5)
func (rl *Relay) handleDeleteRequest(c context.T, evt *event.T) (err error) {
	log.T.Ln("running relay method")
	// event deletion -- nip09
	for _, t := range evt.Tags {
		if len(t) >= 2 && t[0] == "e" {
			// first we fetch the event
			for _, query := range rl.QueryEvents {
				var ch = make(chan *event.T)
				if ch, err = query(c, &filter.T{IDs: tag.T{t[1]}}); chk.E(err) {
					continue
				}
				target := <-ch
				if target == nil {
					continue
				}
				// got the event, now check if the user can delete it
				acceptDeletion := target.PubKey == evt.PubKey
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
					err = log.E.Err("blocked: %s", msg)
					log.E.Ln(err)
					return
				}
				// don't try to query this same event again
				break
			}
		}
	}
	return nil
}

// OverrideDelete decides whether to veto a delete event.
//
// Temporarily removing delete functionality until a proper tombstone/indexing
// strategy is devised to filter out these events from database results.
func (rl *Relay) OverrideDelete(c context.T, tgt, del *event.T) (ok bool,
	msg string) {
	log.T.Ln("running relay method")
	ok = false
	return
}
