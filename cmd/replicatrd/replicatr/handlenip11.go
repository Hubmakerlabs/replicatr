package replicatr

import (
	"encoding/json"
	"net/http"
)

func (rl *Relay) HandleNIP11(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/nostr+json")
	info := rl.Info
	for _, ovw := range rl.OverwriteRelayInfo {
		info = ovw(r.Context(), r, info)
	}
	rl.E.Chk(json.NewEncoder(w).Encode(info))
}
