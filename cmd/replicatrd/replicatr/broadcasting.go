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

		subs.Range(func(id string, listener *Listener) bool {
			log.D.F("sending event to subscriber %v '%s'", id, evt.ToObject().String())
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
