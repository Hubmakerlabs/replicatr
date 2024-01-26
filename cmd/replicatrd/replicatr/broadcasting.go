package replicatr

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
)

// BroadcastEvent emits an event to all listeners whose filters' match, skipping all filters and actions
// it also doesn't attempt to store the event or trigger any reactions or callbacks
func (rl *Relay) BroadcastEvent(evt *event.T) {
	listeners.Range(func(ws *WebSocket, subs ListenerMap) bool {

		rl.D.Ln("broadcasting event")
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.filters.Match(evt) {
				return true
			}
			rl.E.Chk(ws.WriteEnvelope(
				&eventenvelope.T{
					SubscriptionID: subscriptionid.T(id),
					Event:          evt},
			))
			return true
		})
		return true
	})
}
