package app

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

func (rl *Relay) handleDeleteRequest(c context.T, evt *event.T) (err error) {
	// event deletion -- nip09
	for _, t := range evt.Tags {
		if len(t) >= 2 && t[0] == "e" {
			// first we fetch the event
			for _, query := range rl.QueryEvents {
				var ch chan *event.T
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
				// but if we have a function to overwrite this outcome, use that instead
				for _, odo := range rl.OverwriteDeletionOutcome {
					acceptDeletion, msg = odo(c, target, evt)
				}
				if acceptDeletion {
					// delete it
					for _, del := range rl.DeleteEvent {
						chk.E(del(c, target))
					}
				} else {
					// fail and stop here
					err = fmt.Errorf("blocked: %s", msg)
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
