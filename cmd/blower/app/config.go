package app

import (
	"os"

	"mleku.dev/git/slog"
)

const Name = "vacuum"

var log, chk = slog.New(os.Stderr)

type Config struct {
	Nsec        string `arg:"env:NSEC" json:"nsec" help:"use the nsec (bech32 encoded) for auth and if given, writes it to configuration and will be loaded afterwards until a new one is given"`
	UploadRelay string `arg:"positional,required" json:"-" help:"the relay that will be flooded with all the events pulled from file"`
	SourceFile  string `arg:"positional,required" json:"-" help:"path to .jsonl file containing events to push to the relay"`
	SeckeyHex   string `arg:"-" json:"-"` // for internal use
	Pause       int    `arg:"-p,--pause" default:"50" help:"time in milliseconds (1/1000th of a second) to wait between requests to adjust for some relays rate limiter regimes"`
	Skip        int    `arg:"-s,--skip" help:"number of events to skip for resume"`
}
