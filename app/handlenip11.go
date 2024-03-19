package app

import (
	"encoding/json"
	"net/http"
)

func (rl *Relay) HandleNIP11(w http.ResponseWriter, r *http.Request) {
	log.T.Ln("running relay method")
	w.Header().Set("Content-Type", "application/nostr+json")
	info := rl.Info
	for _, ovw := range rl.OverwriteRelayInfo {
		info = ovw(r.Context(), r, info)
	}
	chk.E(json.NewEncoder(w).Encode(info))
}
