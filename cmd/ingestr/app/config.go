package app

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
)

const Name = "ingestr"

var log, chk = slog.New(os.Stderr)

type Config struct {
	Nsec          string `arg:"env:NSEC" json:"nsec" help:"use the nsec (bech32 encoded) for auth and if given, writes it to configuration and will be loaded afterwards until a new one is given"`
	DownloadRelay string `arg:"positional,required" json:"-"`
	UploadRelay   string `arg:"positional,required" json:"-"`
	Since         int64  `arg:"-s,--since" json:"-" help:"only query events since this unix timestamp" default:"1640305963"`
	PubkeyHex     string `arg:"-" json:"-"`
	SeckeyHex     string `arg:"-" json:"-"`
	// Kinds         kinds.T `arg:"-k,--kinds,separate" help:"comma separated list of kind numbers to ingest"`
	Limit        int   `arg:"-l,--limit" help:"maximum of number of events to return for each interval" default:"500"`
	Interval     int64 `arg:"-i,--interval" help:"number of seconds per interval of requests" default:"3"`
	Pause        int   `arg:"-p,--pause" default:"100" help:"time in milliseconds to wait between requests"`
	OtherPubkeys tag.T `arg:"-f,--follows,separate" help:"other pubkeys to search for"`
}

var defaultKinds = kinds.T{kind.TextNote}
