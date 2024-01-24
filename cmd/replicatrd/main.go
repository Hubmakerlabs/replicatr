package main

import (
	"net/http"
	"os"

	"github.com/Hubmakerlabs/replicatr/cmd/replicatrd/replicatr"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"mleku.online/git/slog"
)

const appName = "replicatr"

func main() {
	slog.SetLogLevel(slog.Trace)
	rl := replicatr.NewRelay(appName)
	db := &badger.BadgerBackend{Path: "/home/me/.replicatr-badger",
		Log: slog.New(os.Stderr, "replicatr-badger")}
	if err := db.Init(); rl.E.Chk(err) {
		rl.E.F("unable to start database: '%s'", err)
		os.Exit(1)
	}
	rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
	rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
	rl.CountEvents = append(rl.CountEvents, db.CountEvents)
	rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)
	rl.I.Ln("running on :3334")
	rl.E.Chk(http.ListenAndServe(":3334", rl))
}
