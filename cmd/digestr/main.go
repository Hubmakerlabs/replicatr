package main

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/cmd/digestr/app"
	"github.com/alexflint/go-arg"
	"mleku.dev/git/slog"
)

var args app.Config
var log, chk = slog.New(os.Stderr)

func main() {
	arg.MustParse(&args)
	if !args.SkipSetup {
		app.CleanUp()
	}

	if err := app.EventsTest(args.EventAmount, args.Seed); err != nil {
		log.F.F("event test failed: %v", err)
	}
	if !args.SkipSetup {
		app.CleanUp()
	}
}
