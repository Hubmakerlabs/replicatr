package replicatr

func (rl *Relay) handleCountRequest(ctx Ctx, ws *WebSocket,
	filter *Filter) (subtotal int64) {

	// overwrite the filter (for example, to eliminate some kinds or tags that
	// we know we don't support)
	for _, ovw := range rl.OverwriteCountFilter {
		ovw(ctx, filter)
	}
	// then check if we'll reject this filter
	for _, reject := range rl.RejectCountFilter {
		if rej, msg := reject(ctx, filter); rej {
			rl.E.Chk(ws.WriteJSON(NoticeEnvelope(msg)))
			return 0
		}
	}
	// run the functions to count (generally it will be just one)
	var e error
	var res int64
	for _, count := range rl.CountEvents {
		if res, e = count(ctx, filter); rl.E.Chk(e) {
			rl.E.Chk(ws.WriteJSON(NoticeEnvelope(e.Error())))
		}
		subtotal += res
	}
	return
}
