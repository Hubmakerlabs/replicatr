package replicatr

import (
	err "errors"

	event2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
)

func (rl *Relay) handleFilter(ctx Ctx, id string,
	eose *WaitGroup, ws *WebSocket, f *Filter) (e error) {

	defer eose.Done()
	// overwrite the filter (for example, to eliminate some kinds or that we
	// know we don't support)
	for _, ovw := range rl.OverwriteFilter {
		ovw(ctx, f)
	}
	if f.Limit < 0 {
		e = err.New("blocked: filter invalidated")
		rl.E.Chk(e)
		return
	}
	// then check if we'll reject this filter (we apply this after overwriting
	// because we may, for example, remove some things from the incoming filters
	// that we know we don't support, and then if the end result is an empty
	// filter we can just reject it)
	for _, reject := range rl.RejectFilter {
		if rej, msg := reject(ctx, f); rej {
			rl.E.Chk(ws.WriteJSON(&NoticeEnvelope{Text: msg}))
			return err.New(normalize.OKMessage(msg, "blocked"))
		}
	}
	// run the functions to query events (generally just one,
	// but we might be fetching stuff from multiple places)
	eose.Add(len(rl.QueryEvents))
	for _, query := range rl.QueryEvents {
		var ch chan *Event
		if ch, e = query(ctx, f); rl.E.Chk(e) {
			rl.E.Chk(ws.WriteJSON(&NoticeEnvelope{Text: e.Error()}))
			eose.Done()
			continue
		}
		go func(ch chan *Event) {
			for event := range ch {
				for _, ovw := range rl.OverwriteResponseEvent {
					ovw(ctx, event)
				}
				rl.E.Chk(ws.WriteJSON(event2.EventEnvelope{
					SubscriptionID: &id,
					T:              *event,
				}))
			}
			eose.Done()
		}(ch)
	}
	return nil
}

