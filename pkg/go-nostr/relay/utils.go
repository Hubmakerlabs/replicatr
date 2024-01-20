package relay

import (
	"net/url"
	"strings"
)

func IsValidRelayURL(u string) bool {
	parsed, e := url.Parse(u)
	if e != nil {
		return false
	}
	if parsed.Scheme != "wss" && parsed.Scheme != "ws" {
		return false
	}
	if len(strings.Split(parsed.Host, ".")) < 2 {
		return false
	}
	return true
}
