package replicatr

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/noticeenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
)

func (rl *Relay) handleCountRequest(c context.T, ws *WebSocket,
	f *filter.T) (subtotal int64) {

	// overwrite the filter (for example, to eliminate some kinds or tags that
	// we know we don't support)
	for _, ovw := range rl.OverwriteCountFilter {
		ovw(c, f)
	}
	// then check if we'll reject this filter
	for _, reject := range rl.RejectCountFilter {
		if rej, msg := reject(c, f); rej {
			rl.E.Chk(ws.WriteJSON(&noticeenvelope.T{Text: msg}))
			return 0
		}
	}
	// run the functions to count (generally it will be just one)
	var e error
	var res int64
	for _, count := range rl.CountEvents {
		if res, e = count(c, f); rl.E.Chk(e) {
			rl.E.Chk(ws.WriteJSON(&noticeenvelope.T{Text: e.Error()}))
		}
		subtotal += res
	}
	return
}
