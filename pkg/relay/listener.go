package relay

import (
	"context"
	"fmt"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	event2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	filters2 "github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/puzpuzpuz/xsync/v2"
)

var log, fails = log2.GetStd()

type Listener struct {
	filters filters2.T
	cancel  context.CancelCauseFunc
}

var listeners = xsync.NewTypedMapOf[*WebSocket, *xsync.MapOf[string, *Listener]](pointerHasher[WebSocket])

func GetListeningFilters() filters2.T {
	respFilters := make(filters2.T, 0, listeners.Size()*2)
	// here we go through all the existing listeners
	listeners.Range(func(_ *WebSocket, subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(_ string, listener *Listener) bool {
		next:
			for _, listenerFilter := range listener.filters {
				for _, respFilter := range respFilters {
					// check if this filter specifically is already added to respFilters
					if filter.FilterEqual(listenerFilter, respFilter) {
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

func setListener(id subscriptionid.T, ws *WebSocket,
	filters filters2.T, cancel context.CancelCauseFunc) {

	subs, _ := listeners.LoadOrCompute(ws,
		func() *xsync.MapOf[string, *Listener] {
			return xsync.NewMapOf[*Listener]()
		})
	subs.Store(string(id), &Listener{filters: filters, cancel: cancel})
}

// remove a specific subscription id from listeners for a given ws client
// and cancel its specific context
func removeListenerId(ws *WebSocket, id subscriptionid.T) {
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

func notifyListeners(evt *event.T) {
	listeners.Range(func(ws *WebSocket, subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.filters.Match(evt) {
				return true
			}
			var e error
			var sid subscriptionid.T
			sid, e = subscriptionid.New(id)
			log.D.Chk(e)
			log.E.Chk(ws.WriteJSON(&event2.Envelope{SubscriptionID: sid, Event: evt}))
			return true
		})
		return true
	})
}
