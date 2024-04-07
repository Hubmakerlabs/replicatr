package khatru

import (
	"fmt"

	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
)

func (rl *Relay) handleDeleteRequest(ctx context.T, evt *event.T) error {
	// event deletion -- nip09
	for _, tag := range evt.Tags {
		if len(tag) >= 2 && tag[0] == "e" {
			// first we fetch the event
			for _, query := range rl.QueryEvents {
				ch, err := query(ctx, &filter.T{IDs: []string{tag[1]}})
				if err != nil {
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
					acceptDeletion, msg = odo(ctx, target, evt)
				}
				if acceptDeletion {
					// delete it
					for _, del := range rl.DeleteEvent {
						del(ctx, target)
					}
				} else {
					// fail and stop here
					return fmt.Errorf("blocked: %s", msg)
				}

				// don't try to query this same event again
				break
			}
		}
	}

	return nil
}
