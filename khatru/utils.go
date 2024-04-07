package khatru

import (
	"context"

	"github.com/fasthttp/websocket"
	"github.com/sebest/xff"
	"mleku.dev/git/nostr/envelopes/authenvelope"
	"mleku.dev/git/nostr/filters"
)

const (
	wsKey = iota
	subscriptionIdKey
)

func RequestAuth(ctx context.Context) {
	ws := GetConnection(ctx)
	ws.authLock.Lock()
	if ws.Authed == nil {
		ws.Authed = make(chan struct{})
	}
	ws.authLock.Unlock()
	ws.WriteMessage(websocket.TextMessage,
		(&authenvelope.Challenge{Challenge: ws.Challenge}).Bytes())
}

func GetConnection(ctx context.Context) *WebSocket {
	return ctx.Value(wsKey).(*WebSocket)
}

func GetAuthed(ctx context.Context) string {
	return GetConnection(ctx).AuthedPublicKey
}

func GetIP(ctx context.Context) string {
	return xff.GetRemoteAddr(GetConnection(ctx).Request)
}

func GetSubscriptionID(ctx context.Context) string {
	return ctx.Value(subscriptionIdKey).(string)
}

func GetOpenSubscriptions(ctx context.Context) filters.T {
	if subs, ok := listeners.Load(GetConnection(ctx)); ok {
		res := make(filters.T, 0, listeners.Size()*2)
		subs.Range(func(_ string, sub *Listener) bool {
			res = append(res, sub.fs...)
			return true
		})
		return res
	}
	return nil
}
