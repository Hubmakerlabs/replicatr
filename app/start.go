package app

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/fasthttp/websocket"
	"github.com/rs/cors"
)

// Router returns the http multiplexer in the relay.
func (rl *Relay) Router() *http.ServeMux { return rl.serveMux }

// Start creates an http server and starts listening on given host and port.
func (rl *Relay) Start(host string, port int,
	started ...chan bool) (err error) {

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	var ln net.Listener
	if ln, err = net.Listen("tcp", addr); chk.E(err) {
		return
	}
	rl.Addr = ln.Addr().String()
	rl.httpServer = &http.Server{
		Handler:      cors.Default().Handler(rl),
		Addr:         addr,
		WriteTimeout: 2 * time.Second,
		ReadTimeout:  2 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	// notify caller that we're starting
	for _, s := range started {
		log.I.Ln("closing started chans")
		close(s)
	}
	if err = rl.httpServer.Serve(ln); errors.Is(err, http.ErrServerClosed) {
		return nil
	} else if chk.E(err) {
		return
	}
	return
}

// Shutdown sends a websocket close control message to all connected clients.
func (rl *Relay) Shutdown(c context.T) {
	chk.E(rl.httpServer.Shutdown(c))
	rl.clients.Range(func(conn *websocket.Conn, _ struct{}) bool {
		chk.E(conn.WriteControl(websocket.CloseMessage, nil,
			time.Now().Add(time.Second)))
		chk.E(conn.Close())
		rl.clients.Delete(conn)
		return true
	})
}
