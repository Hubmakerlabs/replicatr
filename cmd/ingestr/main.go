package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/Hubmakerlabs/replicatr/cmd/ingestr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/alexflint/go-arg"
	"mleku.dev/git/ec"
	"mleku.dev/git/ec/schnorr"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

var args app.Config

func main() {
	arg.MustParse(&args)
	var err error
	var dataDirBase string
	if dataDirBase, err = os.UserHomeDir(); chk.E(err) {
		fail()
	}
	confFile := filepath.Join(dataDirBase, "."+app.Name+".json")
	var b []byte
	if args.Nsec != "" {
		// if an nsec is given, write it to a file so it doesn't have to be
		// given again
		if _, _, err = bech32encoding.Decode(args.Nsec); chk.E(err) {
			log.E.F("invalid nsec provided '%s'", args.Nsec)
			fail()
		}
		log.I.F("writing configuration to file %s", confFile)
		if b, err = json.MarshalIndent(&args, "", "    "); chk.E(err) {
			fail()
		}
		if _, err = os.Stat(dataDirBase); errors.Is(err, os.ErrNotExist) {
			if chk.E(os.MkdirAll(dataDirBase, 0700)) {
				fail()
			}
			err = nil
		}
		if chk.E(err) {
			fail()
		}
		if err = os.WriteFile(confFile, b, 0700); chk.E(err) {
			fail()
		}
	} else {
		// try to load the configuration file
		if b, err = os.ReadFile(confFile); chk.E(err) {
			log.E.Ln(`
if no nsec is given there must be configuration, easiest way is to give it in
the -n,--nsec option and it will be created so it can be loaded in future until
this is done again
`)
			fail()
		}
		var cfg app.Config
		if err = json.Unmarshal(b, &cfg); chk.E(err) {
			log.E.Ln(`
unable to read configuration file
`)
			fail()
		}
		args.Nsec = cfg.Nsec
	}
	var nsecDecoded any
	if _, nsecDecoded, err = bech32encoding.Decode(args.Nsec); chk.E(err) {
		log.E.F("invalid nsec provided '%s'", args.Nsec)
		fail()
	}
	args.SeckeyHex = nsecDecoded.(string)
	if args.PubkeyHex, err = keys.GetPublicKey(args.SeckeyHex); chk.E(err) {
		fail()
	}
	var pkb []byte
	if pkb, err = hex.Dec(args.PubkeyHex); chk.E(err) {
		fail()
	}
	var pk *ec.PublicKey
	if pk, err = schnorr.ParsePubKey(pkb); chk.E(err) {
		fail()
	}
	var npub string
	if npub, err = bech32encoding.PublicKeyToNpub(pk); chk.E(err) {
		fail()
	}
	log.I.F("will auth using nsec corresponding to %s", npub)
	os.Exit(app.Ingest(&args))
}

func fail() {
	os.Exit(1)
}
