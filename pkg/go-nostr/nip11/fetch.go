package nip11

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
)

// Fetch fetches the NIP-11 Info.
func Fetch(ctx context.T, u string) (info *Info, e error) {
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.F
		ctx, cancel = context.Timeout(ctx, 7*time.Second)
		defer cancel()
	}

	// normalize URL to start with http:// or https://
	if !strings.HasPrefix(u, "http") && !strings.HasPrefix(u, "ws") {
		u = "wss://" + u
	}
	p, e := url.Parse(u)
	if e != nil {
		return nil, fmt.Errorf("cannot parse url: %s", u)
	}
	if p.Scheme == "ws" {
		p.Scheme = "http"
	} else if p.Scheme == "wss" {
		p.Scheme = "https"
	}
	p.Path = strings.TrimRight(p.Path, "/")

	req, e := http.NewRequestWithContext(ctx, http.MethodGet, p.String(), nil)

	// add the NIP-11 header
	req.Header.Add("Accept", "application/nostr+json")

	// send the request
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		return nil, fmt.Errorf("request failed: %w", e)
	}
	defer resp.Body.Close()

	info = &Info{}
	if e := json.NewDecoder(resp.Body).Decode(info); e != nil {
		return nil, fmt.Errorf("invalid json: %w", e)
	}

	return info, nil
}
