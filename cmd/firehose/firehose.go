// Package main provides a configurable, frequency and event/size volume control
// that hoses a relay with events to test garbage collection.
//
// Keeps track of the event IDs it sends and selects them randomly with a bias
// towards already requested events to request them more so the freshness of
// some events rises above that of others, providing a low frequency of access
// set to delete during a GC pass.
//
// Requires a working relay (any that supports standard protocol and web
// sockets).
package main

import (
	"os"
	"strings"

	"github.com/Hubmakerlabs/replicatr/cmd/firehose/app"
	"github.com/Hubmakerlabs/replicatr/pkg/ec/secp256k1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
	"github.com/alexflint/go-arg"
)

var log, chk = slog.New(os.Stderr)

var (
	cfg app.Config
)

func main() {
	var err error
	arg.MustParse(&cfg)
	log.I.S(cfg)
	if cfg.Nsec == "" {
		app.Sec = keys.GeneratePrivateKey()
		var nsec string
		if nsec, err = bech32encoding.HexToNsec(app.Sec); chk.E(err) {
			panic(err)
		}
		log.I.Ln("signing with", nsec)
	} else {
		if strings.HasPrefix(os.Args[2], bech32encoding.NsecHRP) {
			var sec *secp256k1.SecretKey
			if sec, err = bech32encoding.NsecToSecretKey(os.Args[2]); chk.E(err) {
				os.Exit(1)
			}
			app.Sec = bech32encoding.SecretKeyToHex(sec)
		} else {
			app.Sec = os.Args[2]
		}
	}
	if err = cfg.Main(); chk.E(err) {
	}
}
