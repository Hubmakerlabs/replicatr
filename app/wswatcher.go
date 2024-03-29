package app

import (
	"strings"
	"time"

	"github.com/fasthttp/websocket"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/relayws"
)

type watcherParams struct {
	ctx  context.T
	kill func()
	t    *time.Ticker
	ws   *relayws.WebSocket
}

func (rl *Relay) websocketWatcher(p watcherParams) {
	// log.T.Ln("running relay method")
	var err error
	defer p.kill()
	for {
		select {
		case <-rl.Ctx.Done():
			return
		case <-p.ctx.Done():
			return
		case <-p.t.C:
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
				log.T.F("denying access to '%s': dropping message",
					p.ws.RealRemote())
				return
			}
			if err = p.ws.WriteMessage(websocket.PingMessage,
				nil); log.T.Chk(err) {
				if !strings.HasSuffix(err.Error(),
					"use of closed network connection") {
					log.T.F("error writing ping: %v; closing websocket", err)
				}
				return
			}
		}
	}
}
