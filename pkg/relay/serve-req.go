package relay

import (
	"errors"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/OK"
	event2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/notice"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
)

func (rl *Relay) handleRequest(c context.T, id subscriptionid.T,
	eose *sync.WaitGroup, ws *WebSocket, f *filter.T) (e error) {

	defer eose.Done()
	// overwrite the filter (for example, to eliminate some kinds or
	// that we know we don't support)
	for _, ovw := range rl.OverwriteFilter {
		ovw(c, f)
	}
	if f.Limit < 0 {
		return errors.New("blocked: filter invalidated")
	}
	// then check if we'll reject this filter (we apply this after overwriting
	// because we may, for example, remove some things from the incoming filters
	// that we know we don't support, and then if the end result is an empty
	// filter we can just reject it)
	for _, reject := range rl.RejectFilter {
		if reject, msg := reject(c, f); reject {
			rl.D.Chk(ws.WriteJSON(notice.Envelope{Text: msg}))
			return errors.New(OK.Message(OK.Blocked, msg))
		}
	}
	// run the functions to query events (generally just one,
	// but we might be fetching stuff from multiple places)
	eose.Add(len(rl.QueryEvents))
	for _, query := range rl.QueryEvents {
		var ch chan *event.T
		if ch, e = query(c, f); rl.E.Chk(e) {
			rl.D.Chk(ws.WriteJSON(notice.Envelope{Text: e.Error()}))
			eose.Done()
			continue
		}
		go func(ch chan *event.T) {
			for evt := range ch {
				for _, ovw := range rl.OverwriteResponseEvent {
					ovw(c, evt)
				}
				rl.D.Chk(ws.WriteJSON(event2.Envelope{SubscriptionID: id, Event: evt}))
			}
			eose.Done()
		}(ch)
	}
	return nil
}

func (rl *Relay) handleCountRequest(c context.T, ws *WebSocket, f *filter.T) int64 {
	// overwrite the filter (for example, to eliminate some kinds or tags that
	// we know we don't support)
	for _, ovw := range rl.OverwriteCountFilter {
		ovw(c, f)
	}
	// then check if we'll reject this filter
	for _, reject := range rl.RejectCountFilter {
		if rejecting, msg := reject(c, f); rejecting {
			rl.D.Chk(ws.WriteJSON(notice.Envelope{Text: msg}))
			return 0
		}
	}
	// run the functions to count (generally it will be just one)
	var subtotal int64 = 0
	for _, count := range rl.CountEvents {
		var e error
		var res int64
		if res, e = count(c, f); rl.E.Chk(e) {
			rl.D.Chk(ws.WriteJSON(notice.Envelope{Text: e.Error()}))
		}
		subtotal += res
	}
	return subtotal
}
