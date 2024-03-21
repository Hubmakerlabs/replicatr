package app

import (
	"net/http"

	"github.com/rs/cors"
)

// ServeHTTP implements http.Handler interface.
//
// This is the main starting function of the relay. This launches
// HandleWebsocket which runs the message handling main loop.
func (rl *Relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// log.T.Ln("ServeHTTP")
	select {
	case <-rl.Ctx.Done():
		log.W.Ln("shutting down")
		return
	default:
	}
	if rl.ServiceURL.Load() == "" {
		rl.ServiceURL.Store(getServiceBaseURL(r))
	}
	if r.Header.Get("Upgrade") == "websocket" {
		rl.HandleWebsocket(w, r)
	} else if r.Header.Get("Accept") == "application/nostr+json" {
		cors.AllowAll().Handler(http.HandlerFunc(rl.HandleNIP11)).ServeHTTP(w,
			r)
	} else {
		rl.serveMux.ServeHTTP(w, r)
	}
}
