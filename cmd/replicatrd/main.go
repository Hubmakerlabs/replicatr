package main

import (
	"net/http"
	"os"

	"github.com/Hubmakerlabs/replicatr/cmd/replicatrd/replicatr"
	"github.com/Hubmakerlabs/replicatr/pkg/eventstore/badger"
)

const appName = "replicatr"

func main() {
	r := replicatr.NewRelay(appName)

	db := badger.BadgerBackend{Path: "/tmp/replicatr-badger"}
	if err := db.Init(); err != nil {
		r.Log.E.F("unable to start database: '%s'", err)
		os.Exit(1)
	}
	r.StoreEvent = append(r.StoreEvent, db.SaveEvent)
	r.QueryEvents = append(r.QueryEvents, db.QueryEvents)
	r.CountEvents = append(r.CountEvents, db.CountEvents)
	r.DeleteEvent = append(r.DeleteEvent, db.DeleteEvent)
	r.Log.I.Ln("running on :3334")
	r.Log.E.Chk(http.ListenAndServe(":3334", r))
}
