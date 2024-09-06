package main

import (
	"net/url"
	"os"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/ec/secp256k1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

var log, chk = slog.New(os.Stderr)

func main() {
	if len(os.Args) != 3 {
		log.E.Ln("error: two arguments are required: <relay URL> <nsec>, got",
			os.Args)
	}
	u, err := url.Parse(os.Args[1])
	if chk.E(err) {
		os.Exit(1)
	}
	var sec *secp256k1.SecretKey
	var secKeyHex string
	if strings.HasPrefix(os.Args[2], bech32encoding.NsecHRP) {
		if sec, err = bech32encoding.NsecToSecretKey(os.Args[2]); chk.E(err) {
			os.Exit(1)
		}
		secKeyHex = bech32encoding.SecretKeyToHex(sec)
	} else {
		secKeyHex = os.Args[2]
	}
	c := context.Bg()
	var rl *client.T
	if rl, err = client.ConnectWithAuth(c, u.String(), secKeyHex); chk.E(err) {
		os.Exit(1)
	}
	log.I.Ln("connected to relay with auth", rl.AuthEventID)
}
