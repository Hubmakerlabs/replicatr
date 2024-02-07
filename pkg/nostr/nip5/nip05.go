package nip5

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"
	"mleku.online/git/slog"
)

var log = slog.GetStd()

type (
	name2KeyMap   map[string]string
	key2RelaysMap map[string][]string
)

type WellKnownResponse struct {
	Names  name2KeyMap   `json:"names"`  // NIP-05
	Relays key2RelaysMap `json:"relays"` // NIP-35
}

func QueryIdentifier(c context.T, fullname string) (pp *pointers.Profile, err error) {
	spl := strings.Split(fullname, "@")
	var name, domain string
	switch len(spl) {
	case 1:
		name = "_"
		domain = spl[0]
	case 2:
		name = spl[0]
		domain = spl[1]
	default:
		return nil, fmt.Errorf("not a valid nip-05 identifier")
	}
	if strings.Index(domain, ".") == -1 {
		return nil, fmt.Errorf("hostname doesn't have a dot")
	}
	var req *http.Request
	req, err = http.NewRequestWithContext(c, "GET",
		fmt.Sprintf("https://%s/.well-known/nostr.json?name=%s", domain, name), nil)
	if log.E.Chk(err) {
		return nil, fmt.Errorf("failed to create a request: %w", err)
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) (err error) {
			return http.ErrUseLastResponse
		},
	}
	var res *http.Response
	if res, err = client.Do(req); log.E.Chk(err) {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()
	var result WellKnownResponse
	if err = json.NewDecoder(res.Body).Decode(&result); log.E.Chk(err) {
		return nil, fmt.Errorf("failed to decode json response: %w", err)
	}
	pubkey, ok := result.Names[name]
	if !ok {
		return &pointers.Profile{}, nil
	}
	if len(pubkey) == 64 {
		if _, err := hex.Dec(pubkey); err != nil {
			return &pointers.Profile{}, nil
		}
	}
	relays, _ := result.Relays[pubkey]
	return &pointers.Profile{
		PublicKey: pubkey,
		Relays:    relays,
	}, nil
}

func NormalizeIdentifier(fullname string) string {
	if strings.HasPrefix(fullname, "_@") {
		return fullname[2:]
	}
	return fullname
}
