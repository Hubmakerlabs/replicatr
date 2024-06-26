package app

import (
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
)

type watcherParams struct {
	ctx  context.T
	kill func()
	t    *time.Ticker
	ws   *relayws.WebSocket
}

func (rl *Relay) websocketWatcher(p watcherParams) {
	var err error
	// defer p.kill()
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
				// log.T.F("denying access to '%s': dropping message",
				// 	p.ws.RealRemote())
				return
			}
			if err = p.ws.Ping(); err != nil {
				if !strings.HasSuffix(err.Error(),
					"use of closed network connection") {
					// log.T.F("error writing ping: %v; closing websocket", err)
				}
				p.kill()
				return
			}
		}
	}
}
