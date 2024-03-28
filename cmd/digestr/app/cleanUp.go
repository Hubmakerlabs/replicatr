package app

import (
	"context"
	"encoding/gob"
	"os"
	"path/filepath"

	"github.com/Hubmakerlabs/replicatr/pkg/ic/agent"
	"mleku.dev/git/slog"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/alexflint/go-arg"
)

var args app.Config

func CleanUp() error {
	//load canisterID and canisterAddress from config.json
	var log, chk = slog.New(os.Stderr)
	arg.MustParse(&args)
	var dataDirBase string
	var err error
	if dataDirBase, err = os.UserHomeDir(); chk.E(err) {
		return err
	}
	dataDir := filepath.Join(dataDirBase, args.Profile)
	log.D.F("using profile directory: %s", args.Profile)
	configPath := filepath.Join(dataDir, "config.json")
	args.Load(configPath)

	//load context from context.gob
	contextPath := "context.gob"
	file, err := os.Open(contextPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	var c context.Context
	if err := decoder.Decode(&c); err != nil {
		panic(err)
	}

	//use args.CannisterAddr and args.CannisterId to wipe database
	var b *agent.Backend
	if b, err = agent.New(c, args.CanisterID, args.CanisterAddr); chk.E(err) {
		return err
	}

	//clear all events from canister
	b.ClearEvents(nil)

	return nil

}
