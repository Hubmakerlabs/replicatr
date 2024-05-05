package app

import (
	"os"

	"mleku.dev/git/slog"
)

const Name = "ingestr"

var log, chk = slog.New(os.Stderr)

type Config struct {
	Nsec           string `arg:"-n,--nsec" json:"nsec" help:"use the nsec (bech32 encoded) for auth and if given, writes it to configuration and will be loaded afterwards until a new one is given"`
	DownloadRelays string `arg:"-d,--downloadrelay,required" json:"-" help:"add a relay to pull events from"`
	UploadRelay    string `arg:"positional,required" json:"-" help:"the relay that will be flooded with all the events pulled from the download relays [-d/--downloadrelay]"`
	SeckeyHex      string `arg:"-" json:"-"`
	Pause          int    `arg:"-p,--pause" help:"time in seconds to wait between requests"`
}
