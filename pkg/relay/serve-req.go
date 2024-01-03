package relay

import (
	"context"
	"errors"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
)

func (rl *Relay) handleRequest(ctx context.Context, id nip1.SubscriptionID,
	eose *sync.WaitGroup, ws *WebSocket, filter *nip1.Filter) (e error) {

	defer eose.Done()
	// overwrite the filter (for example, to eliminate some kinds or
	// that we know we don't support)
	for _, ovw := range rl.OverwriteFilter {
		ovw(ctx, filter)
	}
	if filter.Limit < 0 {
		return errors.New("blocked: filter invalidated")
	}
	// then check if we'll reject this filter (we apply this after overwriting
	// because we may, for example, remove some things from the incoming filters
	// that we know we don't support, and then if the end result is an empty
	// filter we can just reject it)
	for _, reject := range rl.RejectFilter {
		if reject, msg := reject(ctx, filter); reject {
			rl.Log.D.Chk(ws.WriteJSON(nip1.NoticeEnvelope{Text: msg}))
			return errors.New(nip1.OKMessage(nip1.OKBlocked, msg))
		}
	}
	// run the functions to query events (generally just one,
	// but we might be fetching stuff from multiple places)
	eose.Add(len(rl.QueryEvents))
	for _, query := range rl.QueryEvents {
		ch, err := query(ctx, filter)
		if err != nil {
			rl.Log.D.Chk(ws.WriteJSON(nip1.NoticeEnvelope{Text: err.Error()}))
			eose.Done()
			continue
		}
		go func(ch chan *nip1.Event) {
			for event := range ch {
				for _, ovw := range rl.OverwriteResponseEvent {
					ovw(ctx, event)
				}
				rl.Log.D.Chk(ws.WriteJSON(nip1.EventEnvelope{SubscriptionID: id, Event: event}))
			}
			eose.Done()
		}(ch)
	}
	return nil
}

func (rl *Relay) handleCountRequest(ctx context.Context, ws *WebSocket, filter *nip1.Filter) int64 {
	// overwrite the filter (for example, to eliminate some kinds or tags that we know we don't support)
	for _, ovw := range rl.OverwriteCountFilter {
		ovw(ctx, filter)
	}
	// then check if we'll reject this filter
	for _, reject := range rl.RejectCountFilter {
		if rejecting, msg := reject(ctx, filter); rejecting {
			rl.Log.D.Chk(ws.WriteJSON(nip1.NoticeEnvelope{Text: msg}))
			return 0
		}
	}
	// run the functions to count (generally it will be just one)
	var subtotal int64 = 0
	for _, count := range rl.CountEvents {
		var e error
		var res int64
		if res, e = count(ctx, filter); rl.Log.E.Chk(e) {
			rl.Log.D.Chk(ws.WriteJSON(nip1.NoticeEnvelope{Text: e.Error()}))
		}
		subtotal += res
	}
	return subtotal
}
