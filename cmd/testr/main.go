package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Hubmakerlabs/replicatr/cmd/testr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/replicatr"
	"github.com/alexflint/go-arg"
	"github.com/fiatjaf/eventstore/badger"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/slog"
)

var args app.Config
var log, chk = slog.New(os.Stderr)

func main() {
	arg.MustParse(&args)
	var err error
	c, cancel := context.Cancel(context.Bg())

	relayArgs := []string{"go run", "-I", args.CanisterID, "-C", args.CanisterAddr, "-p", "testDB", "--loglevel", "error", "initcfg"}

	var dataDirBase string
	if dataDirBase, err = os.UserHomeDir(); chk.E(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, "testDB/filterCheckerDB")
	go replicatr.Main(relayArgs, c, cancel)
	defer cancel()
	db := &badger.BadgerBackend{Path: dataDir}
	if err := db.Init(); err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	if !args.SkipSetup {
		app.CanisterCleanUp(args.CanisterID, args.CanisterAddr)
	}

	if err = app.EventsTest(db, args.EventAmount, args.Seed, c); err != nil {
		log.F.F("event test failed: %v", err)
	}
	if !args.SkipSetup {
		app.CanisterCleanUp(args.CanisterID, args.CanisterAddr)
		app.BadgerCleanUp()
	}
}
