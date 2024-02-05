package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/Hubmakerlabs/replicatr/cmd/ingestr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/alexflint/go-arg"
	"mleku.online/git/ec"
	"mleku.online/git/ec/schnorr"
	"mleku.online/git/slog"
)

var log = slog.New(os.Stderr, app.Name)

var args app.Config

func main() {
	arg.MustParse(&args)
	var err error
	var dataDirBase string
	if dataDirBase, err = os.UserHomeDir(); log.E.Chk(err) {
		fail()
	}
	confFile := filepath.Join(dataDirBase, "."+app.Name+".json")
	var b []byte
	if args.Nsec != "" {
		// if an nsec is given, write it to a file so it doesn't have to be
		// given again
		if _, _, err = bech32encoding.Decode(args.Nsec); log.E.Chk(err) {
			log.E.F("invalid nsec provided '%s'", args.Nsec)
			fail()
		}
		log.I.F("writing configuration to file %s", confFile)
		if b, err = json.MarshalIndent(&args, "", "    "); log.E.Chk(err) {
			fail()
		}
		if _, err = os.Stat(dataDirBase); errors.Is(err, os.ErrNotExist) {
			if log.E.Chk(os.MkdirAll(dataDirBase, 0700)) {
				fail()
			}
			err = nil
		}
		if log.E.Chk(err) {
			fail()
		}
		if err = os.WriteFile(confFile, b, 0700); log.E.Chk(err) {
			fail()
		}
	} else {
		// try to load the configuration file
		if b, err = os.ReadFile(confFile); log.Fail(err) {
			log.E.Ln(`
if no nsec is given there must be configuration, easiest way is to give it in
the -n,--nsec option and it will be created so it can be loaded in future until
this is done again
`)
			fail()
		}
		var cfg app.Config
		if err = json.Unmarshal(b, &cfg); log.Fail(err) {
			log.E.Ln(`
unable to read configuration file
`)
			fail()
		}
		args.Nsec = cfg.Nsec
	}
	var nsecDecoded any
	if _, nsecDecoded, err = bech32encoding.Decode(args.Nsec); log.E.Chk(err) {
		log.E.F("invalid nsec provided '%s'", args.Nsec)
		fail()
	}
	args.SeckeyHex = nsecDecoded.(string)
	if args.PubkeyHex, err = keys.GetPublicKey(args.SeckeyHex); log.E.Chk(err) {
		fail()
	}
	var pkb []byte
	if pkb, err = hex.Dec(args.PubkeyHex); log.E.Chk(err) {
		fail()
	}
	var pk *ec.PublicKey
	if pk, err = schnorr.ParsePubKey(pkb); log.E.Chk(err) {
		fail()
	}
	var npub string
	if npub, err = bech32encoding.PublicKeyToNpub(pk); log.E.Chk(err) {
		fail()
	}
	log.I.F("will auth using nsec corresponding to %s", npub)
	os.Exit(app.Ingest(&args))
}

func fail() {
	os.Exit(1)
}
