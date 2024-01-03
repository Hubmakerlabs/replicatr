package replicatr

import (
	"github.com/nbd-wtf/go-nostr"
	"github.com/sebest/xff"
)

const (
	wsKey = iota
	subscriptionIdKey
)

func RequestAuth(ctx Ctx) {
	ws := GetConnection(ctx)
	ws.authLock.Lock()
	if ws.Authed == nil {
		ws.Authed = make(chan struct{})
	}
	ws.authLock.Unlock()
	log.E.Chk(ws.WriteJSON(nostr.AuthEnvelope{Challenge: &ws.Challenge}))
}

func GetConnection(ctx Ctx) *WebSocket { return ctx.Value(wsKey).(*WebSocket) }

func GetAuthed(ctx Ctx) string { return GetConnection(ctx).AuthedPublicKey }

func GetIP(ctx Ctx) string { return xff.GetRemoteAddr(GetConnection(ctx).Request) }

func GetSubscriptionID(ctx Ctx) string { return ctx.Value(subscriptionIdKey).(string) }

func GetOpenSubscriptions(ctx Ctx) Filters {
	if subs, ok := listeners.Load(GetConnection(ctx)); ok {
		res := make(Filters, 0, listeners.Size()*2)
		subs.Range(func(_ string, sub *Listener) bool {
			res = append(res, sub.filters...)
			return true
		})
		return res
	}
	return nil
}
