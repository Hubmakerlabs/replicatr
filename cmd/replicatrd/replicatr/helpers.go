package replicatr

import (
	"encoding/hex"
	"hash/maphash"
	"strconv"
	"strings"
	"unsafe"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sebest/xff"
	log2 "mleku.online/git/log"
)

const (
	wsKey = iota
	subscriptionIdKey
)

var (
	log                    = log2.GetLogger()
	fails                  = log.D.Chk
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
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

func pointerHasher[V any](_ maphash.Seed, k *V) uint64 {
	return uint64(uintptr(unsafe.Pointer(k)))
}

func isOlder(prev, next *Event) bool {
	p, n := prev.CreatedAt, next.CreatedAt
	return p < n || (p == n && prev.ID > next.ID)
}

func getServiceBaseURL(r *Request) string {
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if host == "localhost" {
			proto = "http"
		} else if strings.Index(host, ":") != -1 {
			// has a port number
			proto = "http"
		} else if _, e := strconv.Atoi(strings.ReplaceAll(host, ".", "")); log.E.Chk(e) {
			// it's a naked IP
			proto = "http"
		} else {
			proto = "https"
		}
	}
	return proto + "://" + host
}
