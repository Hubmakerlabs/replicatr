package app

import (
	"net/http"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/gorilla/websocket"
)

type readParams struct {
	c    context.T
	kill func()
	ws   *relayws.WebSocket
	conn *websocket.Conn
	r    *http.Request
}

func (rl *Relay) websocketReadMessages(p readParams) {
	if p.ws.OffenseCount.Load() > IgnoreAfter {
		log.T.Ln("dropping message due to over", IgnoreAfter,
			"errors from this client on this connection",
			p.ws.RealRemote(), p.ws.AuthPubKey())
		return
	}
	deny := true
	if len(rl.Whitelist) > 0 {
		for i := range rl.Whitelist {
			if rl.Whitelist[i] == p.ws.RealRemote() {
				deny = false
			}
		}
	} else {
		deny = false
	}
	if deny {
		// log.T.F("denying access to '%s': dropping message",
		// 	p.ws.RealRemote())
		// p.kill()
		return
	}
	p.conn.SetReadLimit(int64(MaxMessageSize))
	chk.E(p.conn.SetReadDeadline(time.Now().Add(rl.PongWait)))
	p.conn.SetPongHandler(func(string) (err error) {
		err = p.conn.SetReadDeadline(time.Now().Add(rl.PongWait))
		chk.E(err)
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
		if err != nil {
			// log.I.F("%s from %s, %d bytes message", err, p.ws.RealRemote(), len(message))
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,    // 1000
				websocket.CloseGoingAway,        // 1001
				websocket.CloseNoStatusReceived, // 1005
				websocket.CloseAbnormalClosure,  // 1006
			) {
				log.E.F("unexpected close error from %s: %v",
					p.ws.RealRemote(), err)
			}
			return
		}
		if typ == websocket.PingMessage {
			chk.E(p.ws.Pong())
			continue
		}
		// log.I.Ln("received message", string(message), p.ws.RealRemote())
		if err = rl.wsProcessMessages(message, p.c, p.kill, p.ws); err != nil {
		}
	}
}
