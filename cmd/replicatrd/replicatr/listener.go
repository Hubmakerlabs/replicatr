package replicatr

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/puzpuzpuz/xsync/v2"
)

type Listener struct {
	filters nostr.Filters
	cancel  context.CancelCauseFunc
}

var listeners = xsync.NewTypedMapOf[*WebSocket, *xsync.MapOf[string, *Listener]](pointerHasher[WebSocket])

func GetListeningFilters() (respFilters nostr.Filters) {
	respFilters = make(nostr.Filters, 0, listeners.Size()*2)
	// here we go through all the existing listeners
	listeners.Range(func(_ *WebSocket, subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(_ string, listener *Listener) bool {
			for _, listenerFilter := range listener.filters {
				for _, respFilter := range respFilters {
					// check if this filter specifically is already added to respFilters
					if nostr.FilterEqual(listenerFilter, respFilter) {
						goto next
					}
				}
				// field not yet present on respFilters, add it
				respFilters = append(respFilters, listenerFilter)
				// continue to the next filter
			next:
				continue
			}
			return true
		})
		return true
	})
	return
}

func setListener(id string, ws *WebSocket, f Filters, c context.CancelCauseFunc) {
	subs, _ := listeners.LoadOrCompute(ws, func() *xsync.MapOf[string, *Listener] {
		return xsync.NewMapOf[*Listener]()
	})
	subs.Store(id, &Listener{filters: f, cancel: c})
}

// remove a specific subscription id from listeners for a given ws client
// and cancel its specific context
func removeListenerId(ws *WebSocket, id string) {
	if subs, ok := listeners.Load(ws); ok {
		if listener, ok := subs.LoadAndDelete(id); ok {
			listener.cancel(fmt.Errorf("subscription closed by client"))
		}
		if subs.Size() == 0 {
			listeners.Delete(ws)
		}
	}
}

// remove WebSocket conn from listeners
// (no need to cancel contexts as they are all inherited from the main connection context)
func removeListener(ws *WebSocket) { listeners.Delete(ws) }

func notifyListeners(event Event) {
	listeners.Range(func(ws *WebSocket, subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.filters.Match(event) {
				return true
			}
			log.E.Chk(ws.WriteJSON(nostr.EventEnvelope{SubscriptionID: &id, Event: *event}))
			return true
		})
		return true
	})
}
