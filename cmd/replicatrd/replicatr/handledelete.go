package replicatr

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

func (rl *Relay) handleDeleteRequest(c context.T, evt *event.T) (e error) {
	// event deletion -- nip09
	for _, t := range evt.Tags {
		if len(t) >= 2 && t[0] == "e" {
			// first we fetch the event
			for _, query := range rl.QueryEvents {
				var ch chan *event.T
				if ch, e = query(c, &filter.T{IDs: tag.T{t[1]}}); rl.E.Chk(e) {
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
						rl.E.Chk(del(c, target))
					}
				} else {
					// fail and stop here
					e = fmt.Errorf("blocked: %s", msg)
					rl.E.Ln(e)
					return
				}
				// don't try to query this same event again
				break
			}
		}
	}
	return nil
}
