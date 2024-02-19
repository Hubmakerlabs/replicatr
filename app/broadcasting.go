package app

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

// BroadcastEvent emits an event to all listeners whose filters' match, skipping all filters and actions
// it also doesn't attempt to store the event or trigger any reactions or callbacks
func (rl *Relay) BroadcastEvent(evt *event.T) {
	// var remotes []string
	listeners.Range(func(ws *relayws.WebSocket, subs ListenerMap) bool {

		if ws.AuthPubKey.Load() == "" && rl.Info.Limitation.AuthRequired {
			return true
		}
		log.D.Ln("broadcasting", ws.RealRemote.Load(), ws.AuthPubKey.Load(), subs.Size())
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.filters.Match(evt) {
				log.D.F("filter doesn't match subscription %s %s\nfilters\n%s\nevent\n%s",
					listener.ws.RealRemote.Load(), listener.ws.AuthPubKey.Load(),
					listener.filters, evt.ToObject().String())
				return true
			}
			if kinds.IsPrivileged(evt.Kind) {
				if ws.AuthPubKey.Load() == "" {
					log.D.Ln("not broadcasting privileged event to",
						ws.RealRemote.Load(), "not authenticated")
					return true
				}
				parties := tag.T{evt.PubKey}
				pTags := evt.Tags.GetAll("p")
				for i := range pTags {
					parties = append(parties, pTags[i][1])
				}
				if !parties.Contains(ws.AuthPubKey.Load()) {
					log.D.Ln("not broadcasting privileged event to",
						ws.RealRemote.Load(), "not party to event")
					return true
				}
			}
			log.D.F("sending event to subscriber %v %s (%d %s)",
				ws.RealRemote.Load(), ws.AuthPubKey.Load(),
				// evt.ToObject().String(),
				evt.Kind,
				kind.GetString(evt.Kind),
			)
			// remotes = append(remotes, ws.RealRemote.Load())
			chk.E(ws.WriteEnvelope(&eventenvelope.T{
				SubscriptionID: subscriptionid.T(id),
				Event:          evt},
			))
			return true
		})
		return true
	})
}
