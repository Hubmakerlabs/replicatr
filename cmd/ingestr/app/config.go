package app

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"mleku.dev/git/slog"
)

const Name = "ingestr"

var log, chk = slog.New(os.Stderr)

type Config struct {
	Nsec          string  `arg:"-n,--nsec" json:"nsec" help:"use the nsec (bech32 encoded) for auth and if given, writes it to configuration and will be loaded afterwards until a new one is given"`
	DownloadRelay string  `arg:"positional,required" json:"-"`
	UploadRelay   string  `arg:"positional,required" json:"-"`
	Since         int64   `arg:"-s,--since" json:"-" help:"only query events since this unix timestamp"`
	PubkeyHex     string  `arg:"-" json:"-"`
	SeckeyHex     string  `arg:"-" json:"-"`
	Kinds         kinds.T `arg:"-k,--kinds" help:"comma separated list of kind numbers to ingest"`
	Limit         int     `arg:"-l,--limit" help:"maximum of number of events to return for each interval" default:"1000"`
	Interval      int64   `arg:"-i,--interval" help:"number of hours per interval of requests"`
	Pause         int     `arg:"-p,--pause" help:"time in seconds to wait between requests"`
}

var defaultKinds = kinds.T{kind.TextNote}
