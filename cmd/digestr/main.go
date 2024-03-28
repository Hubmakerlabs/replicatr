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
	app.CleanUp()
	app.GenerateEvents()
	app.FeedEvents()
	app.CleanUp()
}
