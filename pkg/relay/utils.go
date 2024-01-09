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

func RequestAuth(ctx context.T) {
	ws := GetConnection(ctx)
	log.D.Chk(ws.WriteJSON(auth.Challenge{Challenge: ws.Challenge}))
}

func GetConnection(ctx context.T) *WebSocket {
	return ctx.Value(WebsocketContextKey).(*WebSocket)
}

func GetAuthed(ctx context.T) string {
	return GetConnection(ctx).AuthedPublicKey
}

func GetIP(ctx context.T) string {
	return xff.GetRemoteAddr(GetConnection(ctx).Request)
}

func GetSubscriptionID(ctx context.T) string {
	return ctx.Value(SubscriptionIDContextKey).(string)
}

func GetOpenSubscriptions(ctx context.T) (res []*filter.T) {
	if subs, ok := listeners.Load(GetConnection(ctx)); ok {
		res = make([]*filter.T, 0, listeners.Size()*2)
		subs.Range(func(_ string, sub *Listener) bool {
			res = append(res, sub.filters...)
			return true
		})
		return res
	}
	return nil
}
