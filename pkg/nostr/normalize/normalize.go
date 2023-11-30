package normalize

import (
	"net/url"
	"strings"
)

// URL normalizes the url and replaces http://, https:// schemes by
// ws://, wss://.
func URL(u string) string {
	if u == "" {
		return ""
	}
	u = strings.TrimSpace(u)
	u = strings.ToLower(u)
	// if prefix isn't specified as http/s or websocket, assume secure
	// websocket and add wss prefix (this is the most common).
	if !(strings.HasPrefix(u, "http://") ||
		strings.HasPrefix(u, "https://") ||
		strings.HasPrefix(u, "ws://") ||
		strings.HasPrefix(u, "wss://")) {
		u = "wss://" + u
	}
	var e error
	var p *url.URL
	p, e = url.Parse(u)
	if e != nil {
		return ""
	}
	// convert http/s to ws/s
	switch p.Scheme {
	case "https":
		p.Scheme = "wss"
	case "http":
		p.Scheme = "ws"
	}
	// remove trailing path slash
	p.Path = strings.TrimRight(p.Path, "/")
	return p.String()
}
