package digestr

import (
	"os"
	"path/filepath"

	"mleku.dev/git/slog"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/alexflint/go-arg"
)

var args app.Config

func cleanUp() {
	var log, chk = slog.New(os.Stderr)
	arg.MustParse(&args)
	var dataDirBase string
	var err error
	if dataDirBase, err = os.UserHomeDir(); chk.E(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, args.Profile)
	log.D.F("using profile directory: %s", args.Profile)
	configPath := filepath.Join(dataDir, "config.json")
	args.Load(configPath)

	//use args.CannisterAddr and args.CannisterId to wipe database

}
