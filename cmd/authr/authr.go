package main

import (
	"net/url"
	"os"
	"strings"
	"time"

	"mleku.dev/git/ec/secp256k1"
	"mleku.dev/git/nostr/bech32encoding"
	"mleku.dev/git/nostr/client"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/slog"
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
	if rl, err = client.Connect(c, u.String()); chk.E(err) {
		os.Exit(1)
	}
	log.I.Ln("connected to download relay")
	select {
	case <-rl.AuthRequired:
		log.T.Ln("authing to down relay")
		if err = rl.Auth(c,
			func(evt *event.T) error {
				return evt.Sign(secKeyHex)
			}); chk.D(err) {
			os.Exit(1)
		}
	case <-time.After(2 * time.Second):
		log.E.Ln("failed to auth")
	}
}
