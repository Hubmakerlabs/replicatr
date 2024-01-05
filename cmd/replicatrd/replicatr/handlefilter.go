package replicatr

import (
	"errors"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	event2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
)

func (rl *Relay) handleFilter(c context.T, id string,
	eose *sync.WaitGroup, ws *WebSocket, f *filter.T) (e error) {

	defer eose.Done()
	// overwrite the filter (for example, to eliminate some kinds or that we
	// know we don't support)
	for _, ovw := range rl.OverwriteFilter {
		ovw(c, f)
	}
	if f.Limit < 0 {
		e = errors.New("blocked: filter invalidated")
		rl.E.Chk(e)
		return
	}
	// then check if we'll reject this filter (we apply this after overwriting
	// because we may, for example, remove some things from the incoming filters
	// that we know we don't support, and then if the end result is an empty
	// filter we can just reject it)
	for _, reject := range rl.RejectFilter {
		if rej, msg := reject(c, f); rej {
			rl.E.Chk(ws.WriteJSON(notice.Envelope(msg)))
			return errors.New(normalize.OKMessage(msg, "blocked"))
		}
	}
	// run the functions to query events (generally just one,
	// but we might be fetching stuff from multiple places)
	eose.Add(len(rl.QueryEvents))
	for _, query := range rl.QueryEvents {
		var ch chan *event.T
		if ch, e = query(c, f); rl.E.Chk(e) {
			rl.E.Chk(ws.WriteJSON(notice.Envelope(e.Error())))
			eose.Done()
			continue
		}
		go func(ch chan *event.T) {
			for ev := range ch {
				for _, ovw := range rl.OverwriteResponseEvent {
					ovw(c, ev)
				}
				rl.E.Chk(ws.WriteJSON(event2.Envelope{
					SubscriptionID: &id,
					T:              *ev,
				}))
			}
			eose.Done()
		}(ch)
	}
	return nil
}
