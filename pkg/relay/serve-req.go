package relay

import (
	"context"
	"errors"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/OK"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
)

func (rl *Relay) handleRequest(ctx context.Context, id subscriptionid.T,
	eose *sync.WaitGroup, ws *WebSocket, f *filter.T) (e error) {

	defer eose.Done()
	// overwrite the filter (for example, to eliminate some kinds or
	// that we know we don't support)
	for _, ovw := range rl.OverwriteFilter {
		ovw(ctx, f)
	}
	if f.Limit < 0 {
		return errors.New("blocked: filter invalidated")
	}
	// then check if we'll reject this filter (we apply this after overwriting
	// because we may, for example, remove some things from the incoming filters
	// that we know we don't support, and then if the end result is an empty
	// filter we can just reject it)
	for _, reject := range rl.RejectFilter {
		if reject, msg := reject(ctx, f); reject {
			rl.D.Chk(ws.WriteJSON(notice.Envelope{Text: msg}))
			return errors.New(OK.Message(OK.Blocked, msg))
		}
	}
	// run the functions to query events (generally just one,
	// but we might be fetching stuff from multiple places)
	eose.Add(len(rl.QueryEvents))
	for _, query := range rl.QueryEvents {
		var ch chan *event.T
		if ch, e = query(ctx, f); rl.E.Chk(e) {
			rl.D.Chk(ws.WriteJSON(notice.Envelope{Text: e.Error()}))
			eose.Done()
			continue
		}
		go func(ch chan *event.T) {
			for evt := range ch {
				for _, ovw := range rl.OverwriteResponseEvent {
					ovw(ctx, evt)
				}
				rl.D.Chk(ws.WriteJSON(event.Envelope{SubscriptionID: id, Event: evt}))
			}
			eose.Done()
		}(ch)
	}
	return nil
}

func (rl *Relay) handleCountRequest(ctx context.Context, ws *WebSocket, f *filter.T) int64 {
	// overwrite the filter (for example, to eliminate some kinds or tags that
	// we know we don't support)
	for _, ovw := range rl.OverwriteCountFilter {
		ovw(ctx, f)
	}
	// then check if we'll reject this filter
	for _, reject := range rl.RejectCountFilter {
		if rejecting, msg := reject(ctx, f); rejecting {
			rl.D.Chk(ws.WriteJSON(notice.Envelope{Text: msg}))
			return 0
		}
	}
	// run the functions to count (generally it will be just one)
	var subtotal int64 = 0
	for _, count := range rl.CountEvents {
		var e error
		var res int64
		if res, e = count(ctx, f); rl.E.Chk(e) {
			rl.D.Chk(ws.WriteJSON(notice.Envelope{Text: e.Error()}))
		}
		subtotal += res
	}
	return subtotal
}
