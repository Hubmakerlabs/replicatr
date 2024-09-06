package app

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closedenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
)

// AuthCheck sends out a request if auth is required (this is an OnConnect
// method). It just asks for auth if enabled, saving the client time waiting
// until after sending a req.
func (rl *Relay) AuthCheck(c context.T) { rl.IsAuthed(c, "connect") }

func (rl *Relay) IsAuthed(c context.T, envType string) bool {
	ws := GetConnection(c)
	if ws == nil {
		panic("how can has no websocket?")
	}
	// if access requires auth, check that auth is present.
	if rl.Info.Limitation.AuthRequired && ws.AuthPubKey() == "" {
		reason := "this relay requires authentication for " + envType
		log.I.Ln(reason)
		chk.E(ws.WriteEnvelope(&closedenvelope.T{
			Reason: normalize.Reason(reason, auth.Required),
		}))
		// send out authorization request
		RequestAuth(c, envType)
		return false
	}
	return true
}
