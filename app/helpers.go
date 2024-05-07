package app

import (
	"hash/maphash"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/authenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayws"
	"github.com/sebest/xff"
	"mleku.dev/git/slog"
)

const (
	wsKey = iota
	subscriptionIdKey
)

var log, chk = slog.New(os.Stderr)

func RequestAuth(c context.T, envType string) {
	ws := GetConnection(c)
	log.D.Ln("requesting auth from", ws.RealRemote(), "for", envType)
	// ws.authLock.Lock()
	// if ws.Authed == nil {
	// 	ws.Authed = make(chan struct{})
	// }
	// ws.authLock.Unlock()
	chk.E(ws.WriteEnvelope(&authenvelope.Challenge{Challenge: ws.Challenge()}))
}

func GetConnection(c context.T) *relayws.WebSocket {
	v, ok := c.Value(wsKey).(*relayws.WebSocket)
	if !ok {
		return nil
	}
	return v
}

func GetAuthed(c context.T) string { return GetConnection(c).AuthPubKey() }

func GetIP(c context.T) string { return xff.GetRemoteAddr(GetConnection(c).Request) }

func GetSubscriptionID(c context.T) string { return c.Value(subscriptionIdKey).(string) }

func GetOpenSubscriptions(c context.T) filters.T {
	if subs, ok := listeners.Load(GetConnection(c)); ok {
		res := make(filters.T, 0, listeners.Size()*2)
		subs.Range(func(_ string, sub *Listener) bool {
			res = append(res, sub.filters...)
			return true
		})
		return res
	}
	return nil
}

func PointerHasher[V any](_ maphash.Seed, k *V) uint64 {
	return uint64(uintptr(unsafe.Pointer(k)))
}

func isOlder(prev, next *event.T) bool {
	p, n := prev.CreatedAt, next.CreatedAt
	return p < n || (p == n && prev.ID > next.ID)
}

func getServiceBaseURL(r *http.Request) string {
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
		} else if _, err := strconv.Atoi(strings.ReplaceAll(host, ".",
			"")); chk.E(err) {
			// it's a naked IP
			proto = "http"
		} else {
			proto = "https"
		}
	}
	return proto + "://" + host
}
