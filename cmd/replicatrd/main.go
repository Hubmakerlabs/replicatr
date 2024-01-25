package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/Hubmakerlabs/replicatr/cmd/replicatrd/replicatr"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/alexflint/go-arg"
	"mleku.online/git/slog"
)

var args struct {
	Listen  string `default:"0.0.0.0:3334"`
	Profile string `default:"replicatr"`
}

func main() {
	arg.MustParse(&args)
	var dataDirBase string
	var err error
	var log = slog.New(os.Stderr, args.Profile)
	if dataDirBase, err = os.UserHomeDir(); log.E.Chk(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, args.Profile)
	log.D.F("using profile directory: '%s", args.Profile)
	rl := replicatr.NewRelay(log)
	rl.Info.AddNIPs(1, 23, 9, 15, 42, 45)
	db := &badger.BadgerBackend{Path: dataDir, Log: log}
	if err = db.Init(); rl.E.Chk(err) {
		rl.E.F("unable to start database: '%s'", err)
		os.Exit(1)
	}
	rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
	rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
	rl.CountEvents = append(rl.CountEvents, db.CountEvents)
	rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)
	rl.I.Ln("listening on", args.Listen)
	rl.E.Chk(http.ListenAndServe(args.Listen, rl))
}
