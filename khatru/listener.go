package khatru

import (
	"context"
	"fmt"

	"github.com/puzpuzpuz/xsync/v3"
	"mleku.dev/git/nostr/envelopes/eventenvelope"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/filters"
	"mleku.dev/git/nostr/subscriptionid"
)

type Listener struct {
	fs     filters.T
	cancel context.CancelCauseFunc
}

var listeners = xsync.NewMapOf[*WebSocket, *xsync.MapOf[string, *Listener]]()

func GetListeningFilters() filters.T {
	respfilters := make(filters.T, 0, listeners.Size()*2)

	// here we go through all the existing listeners
	listeners.Range(func(_ *WebSocket, subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(_ string, listener *Listener) bool {
			for _, listenerfilter := range listener.fs {
				for _, respfilter := range respfilters {
					// check if this filter specifically is already added to respfilters
					if filter.Equal(listenerfilter, respfilter) {
						goto nextconn
					}
				}

				// field not yet present on respfilters, add it
				respfilters = append(respfilters, listenerfilter)

				// continue to the next filter
			nextconn:
				continue
			}

			return true
		})

		return true
	})

	// respfilters will be a slice with all the distinct filter we currently have active
	return respfilters
}

func setListener(id string, ws *WebSocket, filters filters.T, cancel context.CancelCauseFunc) {
	subs, _ := listeners.LoadOrCompute(ws, func() *xsync.MapOf[string, *Listener] {
		return xsync.NewMapOf[string, *Listener]()
	})
	subs.Store(id, &Listener{fs: filters, cancel: cancel})
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
func removeListener(ws *WebSocket) {
	listeners.Delete(ws)
}

func notifyListeners(ev *event.T) {
	listeners.Range(func(ws *WebSocket, subs *xsync.MapOf[string, *Listener]) bool {
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.fs.Match(ev) {
				return true
			}
			ws.WriteJSON((&eventenvelope.T{SubscriptionID: subscriptionid.T(id), Event: ev}).Bytes())
			return true
		})
		return true
	})
}
