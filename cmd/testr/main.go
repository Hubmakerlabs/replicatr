package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Hubmakerlabs/replicatr/cmd/testr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/replicatr"
	"github.com/alexflint/go-arg"
	"github.com/fasthttp/websocket"
	"github.com/fiatjaf/eventstore/badger"
	"mleku.dev/git/slog"
)

var args app.Config
var log, chk = slog.New(os.Stderr)

func main() {
	arg.MustParse(&args)
	var err error
	ctx, cancel := context.Cancel(context.Bg())

	if args.Wipe {
		app.CanisterCleanUp(args.CanisterID, args.CanisterAddr, args.SecKey)
		app.BadgerCleanUp()
		os.Exit(1)
	}

	relayArgs := []string{"go run", "-I", args.CanisterID, "-C", args.CanisterAddr, "-p", "testDB", "--loglevel",
		args.LogLevel, "initcfg"}

	var dataDirBase string
	if dataDirBase, err = os.UserHomeDir(); chk.E(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, "testDB/filterCheckerDB")
	go replicatr.Main(relayArgs, ctx, cancel)
	defer cancel()
	db := &badger.BadgerBackend{Path: dataDir}
	if err = db.Init(); err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	if !args.SkipSetup {
		app.CanisterCleanUp(args.CanisterID, args.CanisterAddr, args.SecKey)
	}
	// Set up WebSocket connection
	wsURL := "ws://127.0.0.1:3334"
	var c *websocket.Conn
	c, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.F.F("Failed to connect to WebSocket: %s", err)
		return
	}
	defer c.Close()
	var ids []string
	var authors []string
	if authors, ids, err = app.EventsTest(db, args.EventAmount, args.Seed, ctx, c); err != nil {
		log.F.F("event test failed: %v", err)
	}

	if err = app.FiltersTest(authors, ids, db, args.QueryAmount, args.Seed, ctx, c); err != nil {
		log.F.F("event test failed: %v", err)
	}
	if !args.SkipSetup {
		app.CanisterCleanUp(args.CanisterID, args.CanisterAddr, args.SecKey)
		app.BadgerCleanUp()
	}
}
