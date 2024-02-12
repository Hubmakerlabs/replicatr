package app

import (
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/fasthttp/websocket"
)

type watcherParams struct {
	c    context.T
	kill func()
	t    *time.Ticker
	ws   *relayws.WebSocket
}

func (rl *Relay) websocketWatcher(p watcherParams) {

	var err error
	defer p.kill()
	for {
		select {
		case <-p.c.Done():
			return
		case <-p.t.C:
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
				return
			}
			if err = p.ws.WriteMessage(websocket.PingMessage, nil); log.E.Chk(err) {
				if !strings.HasSuffix(err.Error(), "use of closed network connection") {
					log.E.F("error writing ping: %v; closing websocket", err)
				}
				return
			}
		}
	}
}
