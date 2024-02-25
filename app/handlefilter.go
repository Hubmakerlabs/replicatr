package app

import (
	"errors"
	"sync"

	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/envelopes/eventenvelope"
	"mleku.dev/git/nostr/envelopes/noticeenvelope"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/kinds"
	"mleku.dev/git/nostr/normalize"
	"mleku.dev/git/nostr/relayws"
	"mleku.dev/git/nostr/subscriptionid"
	"mleku.dev/git/nostr/tag"
)

type handleFilterParams struct {
	c    context.T
	id   subscriptionid.T
	eose *sync.WaitGroup
	ws   *relayws.WebSocket
	f    *filter.T
}

func (rl *Relay) handleFilter(h handleFilterParams) (err error) {

	defer h.eose.Done()
	// overwrite the filter (for example, to eliminate some kinds or that we
	// know we don't support)
	for _, ovw := range rl.OverwriteFilter {
		ovw(h.c, h.f)
	}
	if h.f.Limit != nil && *h.f.Limit < 0 {
		err = errors.New("blocked: filter invalidated")
		log.E.Ln(err)
		return
	}
	// then check if we'll reject this filter (we apply this after overwriting
	// because we may, for example, remove some things from the incoming filters
	// that we know we don't support, and then if the end result is an empty
	// filter we can just reject it)
	for _, reject := range rl.RejectFilter {
		if rej, msg := reject(h.c, h.id, h.f); rej {
			return errors.New(normalize.Reason(msg, "blocked"))
		}
	}
	// run the functions to query events (generally just one, but we might be
	// fetching stuff from multiple places)
	h.eose.Add(len(rl.QueryEvents))
	for _, query := range rl.QueryEvents {
		ch := make(chan *event.T)
		if ch, err = query(h.c, h.f); chk.E(err) {
			chk.E(h.ws.WriteEnvelope(&noticeenvelope.T{Text: err.Error()}))
			h.eose.Done()
			continue
		}
		go func(ch chan *event.T) {
			for ev := range ch {
				// if the event is nil the rest of this loop will panic
				// accessing the nonexistent event's fields
				if ev == nil {
					continue
				}
				for _, ovw := range rl.OverwriteResponseEvent {
					ovw(h.c, ev)
				}
				if kinds.IsPrivileged(ev.Kind) {
					if h.ws.AuthPubKey() == "" {
						log.D.Ln("not broadcasting privileged event to",
							h.ws.RealRemote(), "not authenticated")
						continue
					}
					// check the filter first
					receivers, _ := h.f.Tags["p"]
					receivers2, _ := h.f.Tags["#p"]
					parties := make(tag.T, len(receivers)+len(receivers2)+len(h.f.Authors))
					copy(parties[:len(h.f.Authors)], h.f.Authors)
					copy(parties[len(h.f.Authors):], receivers)
					copy(parties[len(h.f.Authors)+len(receivers):], receivers2)
					// log.D.Ln(h.ws.RealRemote, "parties", parties)
					if !parties.Contains(h.ws.AuthPubKey()) {
						log.D.Ln("not sending privileged event to user "+
							"without matching auth", parties, h.ws.AuthPubKey())
						continue
					}
					// then check the event
					parties = tag.T{ev.PubKey}
					pTags := ev.Tags.GetAll("p")
					for i := range pTags {
						parties = append(parties, pTags[i][1])
					}
					if !parties.Contains(h.ws.AuthPubKey()) {
						log.D.Ln("not broadcasting privileged event to",
							h.ws.RealRemote(), h.ws.AuthPubKey(),
							"not party to event")
						return
					}
				}
				chk.E(h.ws.WriteEnvelope(&eventenvelope.T{
					SubscriptionID: h.id,
					Event:          ev,
				}))
			}
			h.eose.Done()
		}(ch)
	}
	return nil
}
