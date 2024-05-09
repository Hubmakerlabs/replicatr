package app

import (
	"os"

	"mleku.dev/git/slog"
)

const Name = "vacuum"

var log, chk = slog.New(os.Stderr)

type Config struct {
	Nsec          string `arg:"env:NSEC" json:"nsec" help:"use the nsec (bech32 encoded) for auth and if given, writes it to configuration and will be loaded afterwards until a new one is given"`
	DownloadRelay string `arg:"positional,required" json:"-" help:"specify relay to pull events from"`
	UploadRelay   string `arg:"positional,required" json:"-" help:"the relay that will be flooded with all the events pulled from the download relays [-d/--downloadrelay]"`
	SeckeyHex     string `arg:"-" json:"-"` // for internal use
	Pause         int    `arg:"-p,--pause" default:"100" help:"time in milliseconds (1/1000th of a second) to wait between requests to adjust for some relays rate limiter regimes"`
}
