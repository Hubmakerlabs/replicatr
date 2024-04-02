package main

import (
	"os"
	"time"

	"github.com/Hubmakerlabs/replicatr/cmd/digestr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/replicatr"
	"github.com/alexflint/go-arg"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/slog"
)

var args app.Config
var log, chk = slog.New(os.Stderr)

func main() {
	arg.MustParse(&args)
	c, cancel := context.Cancel(context.Bg())
	relayArgs := []string{"go run", "-I", args.CanisterID, "-C", args.CanisterAddr, "-p", "testDB", "--loglevel", "error", "initcfg"}
	go replicatr.Main(relayArgs, c, cancel)
	defer cancel()
	time.Sleep(time.Second)
	if !args.SkipSetup {
		app.CanisterCleanUp(args.CanisterID, args.CanisterAddr)
	}

	if err := app.EventsTest(args.EventAmount, args.Seed); err != nil {
		log.F.F("event test failed: %v", err)
	}
	if !args.SkipSetup {
		app.CanisterCleanUp(args.CanisterID, args.CanisterAddr)
		app.BadgerCleanUp()
	}
}
