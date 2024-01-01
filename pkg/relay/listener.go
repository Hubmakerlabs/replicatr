package relay

import (
	"context"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/puzpuzpuz/xsync/v2"
)

type Listener struct {
	filters nip1.Filters
	cancel  context.CancelCauseFunc
}

var listeners = xsync.NewTypedMapOf[*WebSocket, *xsync.MapOf[string, *Listener]](pointerHasher[WebSocket])

func GetListeningFilters() nip1.Filters {
	respFilters := make(nip1.Filters, 0, listeners.Size()*2)
	// here we go through all the existing listeners
	listeners.Range(func(_ *WebSocket, subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(_ string, listener *Listener) bool {
		next:
			for _, listenerFilter := range listener.filters {
				for _, respFilter := range respFilters {
					// check if this filter specifically is already added to respFilters
					if nip1.FilterEqual(listenerFilter, respFilter) {
						// continue to the next filter
						continue next
					}
				}
				// field not yet present on respFilters, add it
				respFilters = append(respFilters, listenerFilter)
			}
			return true
		})
		return true
	})
	// respFilters will be a slice with all the distinct filter we currently
	// have active
	return respFilters
}

func setListener(id nip1.SubscriptionID, ws *WebSocket,
	filters nip1.Filters, cancel context.CancelCauseFunc) {

	subs, _ := listeners.LoadOrCompute(ws,
		func() *xsync.MapOf[string, *Listener] {

			return xsync.NewMapOf[*Listener]()
		})
	subs.Store(string(id), &Listener{filters: filters, cancel: cancel})
}

// remove a specific subscription id from listeners for a given ws client
// and cancel its specific context
func removeListenerId(ws *WebSocket, id nip1.SubscriptionID) {
	if subs, ok := listeners.Load(ws); ok {
		if listener, ok := subs.LoadAndDelete(string(id)); ok {
			listener.cancel(fmt.Errorf("subscription closed by client"))
		}
		if subs.Size() == 0 {
			listeners.Delete(ws)
		}
	}
}

// remove WebSocket conn from listeners
// (no need to cancel contexts as they are all inherited from the main connection context)
func removeListener(ws *WebSocket) {
	listeners.Delete(ws)
}

func notifyListeners(event *nip1.Event) {
	listeners.Range(func(ws *WebSocket, subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.filters.Match(event) {
				return true
			}
			var err error
			var sid nip1.SubscriptionID
			sid, err = nip1.NewSubscriptionID(id)
			log.D.Chk(err)
			ws.WriteJSON(nip1.EventEnvelope{SubscriptionID: sid, Event: event})
			return true
		})
		return true
	})
}
