package khatru

import (
	"context"
	"errors"
	"sync"

	"github.com/fasthttp/websocket"
	"mleku.dev/git/nostr/envelopes/eventenvelope"
	"mleku.dev/git/nostr/envelopes/noticeenvelope"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/normalize"
	"mleku.dev/git/nostr/subscriptionid"
)

func (rl *Relay) handleRequest(ctx context.Context, id string, eose *sync.WaitGroup, ws *WebSocket,
	f filter.T) error {
	defer eose.Done()

	// overwrite the filter (for example, to eliminate some kinds or
	// that we know we don't support)
	for _, ovw := range rl.OverwriteFilter {
		ovw(ctx, &f)
	}

	if f.Limit != nil && *f.Limit < 0 {
		// this is a special situation through which the implementor signals to us that it doesn't want
		// to event perform any queries whatsoever
		return nil
	}

	// then check if we'll reject this filter (we apply this after overwriting
	// because we may, for example, remove some things from the incoming filters
	// that we know we don't support, and then if the end result is an empty
	// filter we can just reject it)
	for _, reject := range rl.RejectFilter {
		if reject, msg := reject(ctx, f); reject {
			ws.WriteMessage(websocket.TextMessage, (&noticeenvelope.T{Text: msg}).Bytes())
			return errors.New(normalize.Reason(msg, "blocked"))
		}
	}

	// run the functions to query events (generally just one,
	// but we might be fetching stuff from multiple places)
	eose.Add(len(rl.QueryEvents))
	for _, query := range rl.QueryEvents {
		ch, err := query(ctx, &f)
		if err != nil {
			ws.WriteMessage(websocket.TextMessage, (&noticeenvelope.T{Text: err.Error()}).Bytes())
			eose.Done()
			continue
		}

		go func(ch chan *event.T) {
			for ev := range ch {
				for _, ovw := range rl.OverwriteResponseEvent {
					ovw(ctx, ev)
				}
				ws.WriteMessage(websocket.TextMessage,
					(&eventenvelope.T{
						SubscriptionID: subscriptionid.T(id),
						Event:          ev,
					}).ToArray().Bytes())
			}
			eose.Done()
		}(ch)
	}

	return nil
}

func (rl *Relay) handleCountRequest(ctx context.Context, ws *WebSocket, f filter.T) int {
	// overwrite the filter (for example, to eliminate some kinds or tags that we know we don't support)
	for _, ovw := range rl.OverwriteCountFilter {
		ovw(ctx, &f)
	}

	// then check if we'll reject this filter
	for _, reject := range rl.RejectCountFilter {
		if rejecting, msg := reject(ctx, f); rejecting {
			ws.WriteMessage(websocket.TextMessage, noticeenvelope.NewNoticeEnvelope(msg).Bytes())
			return 0
		}
	}

	// run the functions to count (generally it will be just one)
	var subtotal int
	for _, count := range rl.CountEvents {
		res, err := count(ctx, &f)
		if err != nil {
			ws.WriteMessage(websocket.TextMessage, (&noticeenvelope.T{Text: err.Error()}).Bytes())
		}
		subtotal += res
	}

	return subtotal
}
