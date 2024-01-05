package replicatr

import (
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
)

// BroadcastEvent emits an event to all listeners whose filters' match, skipping all filters and actions
// it also doesn't attempt to store the event or trigger any reactions or callbacks
func (rl *Relay) BroadcastEvent(evt *Event) {
	listeners.Range(func(ws *WebSocket, subs ListenerMap) bool {
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.filters.Match(evt) {
				return true
			}
			log.E.Chk(ws.WriteJSON(event.Envelope{
				SubscriptionID: &id,
				T:              *evt},
			))
			return true
		})
		return true
	})
}
