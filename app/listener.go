package app

import (
	"fmt"

	"github.com/puzpuzpuz/xsync/v2"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/filters"
	"mleku.dev/git/nostr/relayws"
)

type Listener struct {
	filters filters.T
	cancel  context.C
	ws      *relayws.WebSocket
}

type ListenerMap = *xsync.MapOf[string, *Listener]

var listeners = xsync.NewTypedMapOf[*relayws.WebSocket,
	ListenerMap](PointerHasher[relayws.WebSocket])

func GetListeningFilters() (respFilters filters.T) {
	respFilters = make(filters.T, 0, listeners.Size()*2)
	// here we go through all the existing listeners
	listeners.Range(func(_ *relayws.WebSocket, subs ListenerMap) bool {
		subs.Range(func(_ string, listener *Listener) bool {
			for _, listenerFilter := range listener.filters {
				for _, respFilter := range respFilters {
					// check if this filter specifically is already added to
					// respFilters
					if filter.Equal(listenerFilter, respFilter) {
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

func SetListener(id string, ws *relayws.WebSocket, f filters.T, c context.C) {
	subs, _ := listeners.LoadOrCompute(ws, func() ListenerMap {
		return xsync.NewMapOf[*Listener]()
	})
	subs.Store(id, &Listener{filters: f, cancel: c, ws: ws})
}

// RemoveListenerId removes a specific subscription id from listeners for a
// given ws client and cancel its specific context
func RemoveListenerId(ws *relayws.WebSocket, id string) {
	if subs, ok := listeners.Load(ws); ok {
		if listener, ok := subs.LoadAndDelete(id); ok {
			listener.cancel(fmt.Errorf("subscription closed by client"))
		}
		if subs.Size() == 0 {
			listeners.Delete(ws)
		}
	}
}

// RemoveListener removes WebSocket conn from listeners (no need to cancel
// contexts as they are all inherited from the main connection context)
func RemoveListener(ws *relayws.WebSocket) { listeners.Delete(ws) }
