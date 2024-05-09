package app

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

// BroadcastEvent emits an event to all listeners whose filters' match, skipping all filters and actions
// it also doesn't attempt to store the event or trigger any reactions or callbacks
func (rl *Relay) BroadcastEvent(ev *event.T) {
	listeners.Range(func(ws *relayws.WebSocket, subs ListenerMap) bool {

		if ws.AuthPubKey() == "" && rl.Info.Limitation.AuthRequired {
			log.E.Ln("cannot broadcast to", ws.RealRemote(), "not authorized")
			return true
		}
		// log.T.Ln("broadcasting", ws.RealRemote(), ws.AuthPubKey(), subs.Size())
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.filters.Match(ev) {
				// log.T.F("filter doesn't match subscription %s %s\nfilters\n%s\nevent\n%s",
				// 	listener.ws.RealRemote(), listener.ws.AuthPubKey(),
				// 	text.Trunc(listener.filters.ToArray().String()), text.Trunc(ev.ToObject().String()))
				return true
			}
			if kinds.IsPrivileged(ev.Kind) && rl.Info.Limitation.AuthRequired {
				if ws.AuthPubKey() == "" {
					log.T.Ln("not broadcasting privileged event to",
						ws.RealRemote(), "not authenticated")
					return true
				}
				parties := tag.T{ev.PubKey}
				pTags := ev.Tags.GetAll("p")
				for i := range pTags {
					parties = append(parties, pTags[i][1])
				}
				if !parties.Contains(ws.AuthPubKey()) {
					log.T.Ln("not broadcasting privileged event to",
						ws.RealRemote(), "not party to event")
					return true
				}
			}
			log.D.F("sending event to subscriber %v %s (%d %s)",
				ws.RealRemote(), ws.AuthPubKey(),
				ev.Kind,
				ev.Kind.Name(),
			)
			chk.E(ws.WriteEnvelope(&eventenvelope.T{
				SubscriptionID: subscriptionid.T(id),
				Event:          ev},
			))
			return true
		})
		return true
	})
}
