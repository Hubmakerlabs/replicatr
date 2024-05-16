package app

import (
	"errors"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/noticeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

type handleFilterParams struct {
	c    context.T
	id   subscriptionid.T
	eose *sync.WaitGroup
	ws   *relayws.WebSocket
	f    *filter.T
}

func (rl *Relay) handleFilter(h handleFilterParams) (err error) {
	h.eose.Add(1)
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
			return log.D.Err(normalize.Reason(msg, "blocked"))
		}
	}
	// run the functions to query events (generally just one, but we might be
	// fetching stuff from multiple places)
	for _, query := range rl.QueryEvents {
		h.eose.Add(1)
		var ch event.C
		// start up event receiver before running query on this channel
		var kindStrings []string
		if h.f.Kinds != nil && len(h.f.Kinds) > 0 {
			for _, ks := range h.f.Kinds {
				kindStrings = append(kindStrings, kind.GetString(ks))
			}
		}
		if ch, err = query(h.c, h.f); chk.E(err) {
			h.ws.OffenseCount.Inc()
			chk.E(h.ws.WriteEnvelope(&noticeenvelope.T{Text: err.Error()}))
			h.eose.Done()
			continue
		}
		go func(ch event.C) {
			for {
				select {
				case ev := <-ch:
					// if the event is nil the rest of this loop will panic
					// accessing the nonexistent event's fields
					if ev == nil {
						return
					}
					for _, ovw := range rl.OverwriteResponseEvent {
						ovw(h.c, ev)
					}
					if kinds.IsPrivileged(ev.Kind) && rl.Info.Limitation.AuthRequired {
						var allow bool
						for _, v := range rl.Config.AllowIPs {
							if h.ws.RealRemote() == v {
								allow = true
								break
							}
						}
						if h.ws.AuthPubKey() == "" && !allow {
							log.D.Ln("not broadcasting privileged event to",
								h.ws.RealRemote(), "not authenticated")
							continue
						}
						if !allow {
							// check the filter first
							receivers, _ := h.f.Tags["p"]
							receivers2, _ := h.f.Tags["#p"]
							parties := make(tag.T,
								len(receivers)+len(receivers2)+len(h.f.Authors))
							copy(parties[:len(h.f.Authors)], h.f.Authors)
							copy(parties[len(h.f.Authors):], receivers)
							copy(parties[len(h.f.Authors)+len(receivers):],
								receivers2)
							// then check the event
							parties = tag.T{ev.PubKey}
							pTags := ev.Tags.GetAll("p")
							for i := range pTags {
								parties = append(parties, pTags[i][1])
							}
							if !parties.Contains(h.ws.AuthPubKey()) &&
								rl.Info.Limitation.AuthRequired {
								log.D.Ln("not broadcasting privileged event to",
									h.ws.RealRemote(), h.ws.AuthPubKey(),
									"not party to event")
								return
							}
						}
					}
					chk.E(h.ws.WriteEnvelope(&eventenvelope.T{
						SubscriptionID: h.id,
						Event:          ev,
					}))
				case <-rl.Ctx.Done():
					log.T.Ln("shutting down")
					return
				case <-h.c.Done():
					return
				}
			}
		}(ch)
	}
	return nil
}
