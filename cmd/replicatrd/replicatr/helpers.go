package replicatr

import (
	"encoding/hex"
	"hash/maphash"
	"strconv"
	"strings"
	"unsafe"

	log2 "mleku.online/git/log"
)

var (
	log                    = log2.GetLogger()
	fails                  = log.D.Chk
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

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
