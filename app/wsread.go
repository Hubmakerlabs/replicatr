package app

import (
	"net/http"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/fasthttp/websocket"
)

type readParams struct {
	c    context.T
	kill func()
	ws   *relayws.WebSocket
	conn *websocket.Conn
	r    *http.Request
}

func (rl *Relay) websocketReadMessages(p readParams) {

	deny := true
	if len(rl.Whitelist) > 0 {
		for i := range rl.Whitelist {
			if rl.Whitelist[i] == p.ws.RealRemote.Load() {
				deny = false
			}
		}
	} else {
		deny = false
	}
	if deny {
		log.T.F("denying access to '%s': dropping message",
			p.ws.RealRemote.Load())
		// p.kill()
		return
	}
	p.conn.SetReadLimit(rl.MaxMessageSize)
	log.E.Chk(p.conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
	p.conn.SetPongHandler(func(string) (err error) {
		err = p.conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		log.E.Chk(err)
		return
	})
	for _, onConnect := range rl.OnConnect {
		onConnect(p.c)
	}
	for {
		var err error
		var typ int
		var message []byte
		typ, message, err = p.conn.ReadMessage()
		if log.D.Chk(err) {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,    // 1000
				websocket.CloseGoingAway,        // 1001
				websocket.CloseNoStatusReceived, // 1005
				websocket.CloseAbnormalClosure,  // 1006
			) {
				log.E.F("unexpected close error from %s: %v",
					p.ws.RealRemote.Load(), err)
			}
			return
		}
		if typ == websocket.PingMessage {
			chk.E(p.ws.WriteMessage(websocket.PongMessage, nil))
			continue
		}
		log.T.F("receiving message from %s %s: %s",
			p.ws.RealRemote.Load(), p.ws.AuthPubKey.Load(), string(message))
		rl.wsProcessMessages(message, p.c, p.kill, p.ws)
	}
}
