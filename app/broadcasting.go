package app

import (
	"mleku.dev/git/nostr/envelopes/eventenvelope"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/kinds"
	"mleku.dev/git/nostr/relayws"
	"mleku.dev/git/nostr/subscriptionid"
	"mleku.dev/git/nostr/tag"
)

// BroadcastEvent emits an event to all listeners whose filters' match, skipping all filters and actions
// it also doesn't attempt to store the event or trigger any reactions or callbacks
func (rl *Relay) BroadcastEvent(evt *event.T) {
	listeners.Range(func(ws *relayws.WebSocket, subs ListenerMap) bool {

		if ws.AuthPubKey() == "" && rl.Info.Limitation.AuthRequired {
			log.E.Ln("cannot broadcast to", ws.RealRemote(), "not authorized")
			return true
		}
		log.T.Ln("broadcasting", ws.RealRemote(), ws.AuthPubKey(), subs.Size())
		subs.Range(func(id string, listener *Listener) bool {
			if !listener.filters.Match(evt) {
				log.T.F("filter doesn't match subscription %s %s\nfilters\n%s\nevent\n%s",
					listener.ws.RealRemote(), listener.ws.AuthPubKey(),
					listener.filters, evt.ToObject().String())
				return true
			}
			if kinds.IsPrivileged(evt.Kind) && rl.Info.Limitation.AuthRequired {
				if ws.AuthPubKey() == "" {
					log.T.Ln("not broadcasting privileged event to",
						ws.RealRemote(), "not authenticated")
					return true
				}
				parties := tag.T{evt.PubKey}
				pTags := evt.Tags.GetAll("p")
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
				evt.Kind,
				kind.GetString(evt.Kind),
			)
			chk.E(ws.WriteEnvelope(&eventenvelope.T{
				SubscriptionID: subscriptionid.T(id),
				Event:          evt},
			))
			return true
		})
		return true
	})
}
