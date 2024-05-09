package app

import (
	"os"

	"mleku.dev/git/slog"
)

const Name = "vacuum"

var log, chk = slog.New(os.Stderr)

type Config struct {
	Nsec        string `arg:"env:NSEC" json:"nsec" help:"use the nsec (bech32 encoded) for auth and if given, writes it to configuration and will be loaded afterwards until a new one is given"`
	UploadRelay string `arg:"positional,required" json:"-" help:"the relay that will be flooded with all the events pulled from the download relays [-d/--downloadrelay]"`
	SourceFile  string `arg:"positional,required" json:"-" help:"path to .jsonl file containing events to push to the relay"`
	SeckeyHex   string `arg:"-" json:"-"` // for internal use
}
