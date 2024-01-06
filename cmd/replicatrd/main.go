package main

import (
	"net/http"
	"os"

	"github.com/Hubmakerlabs/replicatr/cmd/replicatrd/replicatr"
	"github.com/Hubmakerlabs/replicatr/pkg/eventstore/badger"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
)

const appName = "replicatr"

func main() {
	log2.SetLogLevel(log2.Trace)
	rl := replicatr.NewRelay(appName)
	db := &badger.BadgerBackend{Path: "/tmp/replicatr-badger"}
	if e := db.Init(); rl.E.Chk(e) {
		rl.E.F("unable to start database: '%s'", e)
		os.Exit(1)
	}
	rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
	rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
	rl.CountEvents = append(rl.CountEvents, db.CountEvents)
	rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)
	rl.I.Ln("running on :3334")
	rl.E.Chk(http.ListenAndServe(":3334", rl))
}
