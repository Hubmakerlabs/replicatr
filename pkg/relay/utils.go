package relay

import (
	"context"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip42"
	"github.com/sebest/xff"
)

const (
	WebsocketContextKey = iota
	SubscriptionIDContextKey
)

func RequestAuth(ctx context.Context) {
	ws := GetConnection(ctx)
	log.D.Chk(ws.WriteJSON(nip42.AuthChallengeEnvelope{Challenge: ws.Challenge}))
}

func GetConnection(ctx context.Context) *WebSocket {
	return ctx.Value(WebsocketContextKey).(*WebSocket)
}

func GetAuthed(ctx context.Context) string {
	return GetConnection(ctx).AuthedPublicKey
}

func GetIP(ctx context.Context) string {
	return xff.GetRemoteAddr(GetConnection(ctx).Request)
}

func GetSubscriptionID(ctx context.Context) string {
	return ctx.Value(SubscriptionIDContextKey).(string)
}

func GetOpenSubscriptions(ctx context.Context) (res []*filter.T) {
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
