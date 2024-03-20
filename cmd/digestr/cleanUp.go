package digestr

import (
	"os"
	"path/filepath"

	"github.com/Hubmakerlabs/replicatr/pkg/ic/agent"
	"mleku.dev/git/slog"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/alexflint/go-arg"
)

var args app.Config

func cleanUp() error {
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

	//use args.CannisterAddr and args.CannisterId to wipe database
	var b *agent.Backend
	if b, err = agent.New(nil, args.CanisterID, args.CanisterAddr); chk.E(err) {
		return err
	}

	b.ClearEvents(nil)

	return nil

}
