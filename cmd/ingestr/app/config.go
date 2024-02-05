package app

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"mleku.online/git/slog"
)

const Name = "ingestr"

var log = slog.New(os.Stderr)

type Config struct {
	Nsec          string  `arg:"-n,--nsec" json:"nsec" help:"use the nsec (bech32 encoded) for auth and if given, writes it to configuration and will be loaded afterwards until a new one is given"`
	DownloadRelay string  `arg:"positional,required" json:"-"`
	UploadRelay   string  `arg:"positional,required" json:"-"`
	Since         int64   `arg:"-s,--since" json:"-" help:"only query events since this unix timestamp"`
	PubkeyHex     string  `arg:"-" json:"-"`
	SeckeyHex     string  `arg:"-" json:"-"`
	Kinds         kinds.T `arg:"-k,--kinds" help:"comma separated list of kind numbers to ingest"`
	Limit         int     `arg:"-l,--limit" help:"maximum of number of events to return for each hour long interval" default:"1000"`
}

var defaultKinds = kinds.T{kind.TextNote}
