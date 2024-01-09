package relay

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/sebest/xff"
)

const (
	WebsocketContextKey = iota
	SubscriptionIDContextKey
)

func RequestAuth(c context.T) {
	ws := GetConnection(c)
	log.D.Chk(ws.WriteJSON(auth.Challenge{Challenge: ws.Challenge}))
}

func GetConnection(c context.T) *WebSocket {
	return c.Value(WebsocketContextKey).(*WebSocket)
}

func GetAuthed(c context.T) string {
	return GetConnection(c).AuthedPublicKey
}

func GetIP(c context.T) string {
	return xff.GetRemoteAddr(GetConnection(c).Request)
}

func GetSubscriptionID(c context.T) string {
	return c.Value(SubscriptionIDContextKey).(string)
}

func GetOpenSubscriptions(c context.T) (res []*filter.T) {
	if subs, ok := listeners.Load(GetConnection(c)); ok {
		res = make([]*filter.T, 0, listeners.Size()*2)
		subs.Range(func(_ string, sub *Listener) bool {
			res = append(res, sub.filters...)
			return true
		})
		return res
	}
	return nil
}
