package app

import (
	"crypto/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/fasthttp/websocket"
)

func (rl *Relay) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	var err error
	var conn *websocket.Conn
	conn, err = rl.upgrader.Upgrade(w, r, nil)
	if log.E.Chk(err) {
		log.E.F("failed to upgrade websocket: %v", err)
		return
	}
	rl.clients.Store(conn, struct{}{})
	ticker := time.NewTicker(rl.PingPeriod)
	// NIP-42 challenge
	challenge := make([]byte, 8)
	_, err = rand.Read(challenge)
	log.E.Chk(err)
	rem := r.Header.Get("X-Forwarded-For")
	splitted := strings.Split(rem, " ")
	var rr string
	if len(splitted) == 1 {
		rr = splitted[0]
	}
	if len(splitted) == 2 {
		rr = splitted[1]
	}
	// in case upstream doesn't set this or we are directly ristening instead of
	// via reverse proxy or just if the header field is missing, put the
	// connection remote address into the websocket state data.
	if rr == "" {
		rr = r.RemoteAddr
	}
	ws := &relayws.WebSocket{
		Conn:       conn,
		RealRemote: rr,
		Request:    r,
		Challenge:  hex.Enc(challenge),
		Authed:     make(chan struct{}),
	}
	c, cancel := context.Cancel(
		context.Value(
			context.Bg(),
			wsKey, ws,
		),
	)
	kill := func() {
		for _, onDisconnect := range rl.OnDisconnect {
			onDisconnect(c)
		}
		ticker.Stop()
		cancel()
		if _, ok := rl.clients.Load(conn); ok {
			_ = conn.Close()
			rl.clients.Delete(conn)
			RemoveListener(ws)
		}
	}
	go rl.websocketReadMessages(readParams{c, kill, ws, conn, r})
	go rl.websocketWatcher(watcherParams{c, kill, ticker, ws})
}
