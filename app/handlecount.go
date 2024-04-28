package app

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/noticeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
)

func (rl *Relay) handleCountRequest(c context.T, id subscriptionid.T,
	ws *relayws.WebSocket, f *filter.T) (subtotal int) {

	log.T.Ln("running count method")
	// overwrite the filter (for example, to eliminate some kinds or tags that we
	// know we don't support)
	for _, ovw := range rl.OverwriteCountFilter {
		ovw(c, f)
	}
	// then check if we'll reject this filter
	for _, reject := range rl.RejectCountFilter {
		if rej, msg := reject(c, id, f); rej {
			chk.E(ws.WriteEnvelope(&noticeenvelope.T{Text: msg}))
			return 0
		}
	}
	// run the functions to count (generally it will be just one)
	var err error
	var res int
	for _, count := range rl.CountEvents {
		if res, err = count(c, f); chk.E(err) {
			chk.E(ws.WriteEnvelope(&noticeenvelope.T{Text: err.Error()}))
		}
		subtotal += res
	}
	return
}
