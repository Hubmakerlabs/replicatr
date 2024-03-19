package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/relayws"
)

func (rl *Relay) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	var err error
	var conn *websocket.Conn
	conn, err = rl.upgrader.Upgrade(w, r, nil)
	if chk.E(err) {
		log.E.F("failed to upgrade websocket: %v", err)
		return
	}
	rl.clients.Store(conn, struct{}{})
	ticker := time.NewTicker(rl.PingPeriod)

	chk.E(err)
	rem := r.Header.Get("X-Forwarded-For")
	splitted := strings.Split(rem, " ")
	var rr string
	if len(splitted) == 1 {
		rr = splitted[0]
	}
	if len(splitted) == 2 {
		rr = splitted[1]
	}
	// in case upstream doesn't set this or we are directly listening instead of
	// via reverse proxy or just if the header field is missing, put the
	// connection remote address into the websocket state data.
	if rr == "" {
		rr = r.RemoteAddr
	}
	ws := &relayws.WebSocket{
		Conn:    conn,
		Request: r,
		Authed:  make(chan struct{}),
	}
	ws.SetRealRemote(rr)

	// NIP-42 challenge
	ws.GenerateChallenge()
	c, cancel := context.Cancel(
		context.Value(
			context.Bg(),
			wsKey, ws,
		),
	)
	if len(rl.Whitelist) > 0 {
		for i := range rl.Whitelist {
			if rr == rl.Whitelist[i] {
				log.T.Ln("inbound connection from", rr)
			}
		}
	} else {
		log.T.Ln("inbound connection from", rr)
	}
	kill := func() {
		log.T.Ln("disconnecting websocket", rr)
		for _, onDisconnect := range rl.OnDisconnect {
			onDisconnect(c)
		}
		ticker.Stop()
		cancel()
		if _, ok := rl.clients.Load(conn); ok {
			rl.clients.Delete(conn)
			RemoveListener(ws)
		}
	}
	go rl.websocketReadMessages(readParams{c, kill, ws, conn, r})
	go rl.websocketWatcher(watcherParams{c, kill, ticker, ws})
}
