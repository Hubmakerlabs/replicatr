package relayinfo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

// Fetch fetches the NIP-11 Info.
func Fetch(c context.T, u string) (info *T, err error) {
	if _, ok := c.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		c, cancel = context.Timeout(c, 7*time.Second)
		defer cancel()
	}

	// normalize URL to start with http:// or https://
	if !strings.HasPrefix(u, "http") && !strings.HasPrefix(u, "ws") {
		u = "wss://" + u
	}
	p, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("cannot parse url: %s", u)
	}
	if p.Scheme == "ws" {
		p.Scheme = "http"
	} else if p.Scheme == "wss" {
		p.Scheme = "https"
	}
	p.Path = strings.TrimRight(p.Path, "/")

	req, err := http.NewRequestWithContext(c, http.MethodGet, p.String(), nil)

	// add the NIP-11 header
	req.Header.Add("Accept", "application/nostr+json")

	// send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	var b []byte
	b, err = io.ReadAll(resp.Body)
	// log.I.Ln(string(b))
	// var inf any
	// if err = json.NewDecoder(resp.Body).Decode(info); err != nil {
	// 	return nil, fmt.Errorf("invalid json: %w", err)
	// }
	// log.I.S(inf)
	info = &T{}
	if err = json.Unmarshal(b, info); chk.E(err) {
		return
	}
	// if err = json.NewDecoder(resp.Body).Decode(info); err != nil {
	// 	return nil, fmt.Errorf("invalid json: %w", err)
	// }

	return info, nil
}
