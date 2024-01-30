package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/Hubmakerlabs/replicatr/cmd/replicatrd/replicatr"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip11"
	"github.com/alexflint/go-arg"
	"mleku.online/git/slog"
)

var (
	AppName = "replicatr"
	Version = "v0.0.1"
)

type ExportCmd struct {
	ToFile string `arg:"-f,--tofile" help:"write to file instead of stdout"`
}

type ImportCmd struct {
	FromFile []string `arg:"-f,--fromfile,separate" help:"read from files instead of stdin (can use flag repeatedly for multiple files)"`
}

type Args struct {
	ExportCmd   *ExportCmd `json:"-" arg:"subcommand:export" help:"export database as line structured JSON"`
	ImportCmd   *ImportCmd `json:"-" arg:"subcommand:import" help:"import data from line structured JSON"`
	Listen      string     `json:"listen" arg:"-l,--listen" default:"0.0.0.0:3334" help:"network address to listen on"`
	Profile     string     `json:"-" arg:"-p,--profile" default:"replicatr" help:"profile name to use for storage"`
	Name        string     `json:"name" arg:"-n,--name" default:"replicatr relay" help:"name of relay for NIP-11"`
	Description string     `json:"description" arg:"--description" help:"description of relay for NIP-11"`
	Pubkey      string     `json:"pubkey" arg:"-k,--pubkey" help:"public key of relay operator"`
	Contact     string     `json:"contact" arg:"-c,--contact" help:"non-nostr relay operator contact details"`
	Icon        string     `json:"icon" arg:"-i,--icon" default:"https://i.nostr.build/n8vM.png" help:"icon to show on relay information pages"`
}

var args Args

func main() {
	arg.MustParse(&args)
	var dataDirBase string
	var err error
	var log = slog.New(os.Stderr, args.Profile)
	if dataDirBase, err = os.UserHomeDir(); log.E.Chk(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, args.Profile)
	log.D.F("using profile directory: '%s", args.Profile)
	rl := replicatr.NewRelay(log, &nip11.Info{
		Name:        args.Name,
		Description: args.Description,
		PubKey:      args.Pubkey,
		Contact:     args.Contact,
		Software:    AppName,
		Version:     Version,
		Limitation: &nip11.Limits{
			MaxMessageLength: replicatr.MaxMessageSize,
		},
		RelayCountries: nil,
		LanguageTags:   nil,
		Tags:           nil,
		PostingPolicy:  "",
		PaymentsURL:    "",
		Fees:           &nip11.Fees{},
		Icon:           args.Icon,
	})
	rl.Info.AddNIPs(
		nip11.BasicProtocol.Number,            // events, envelopes and filters
		nip11.FollowList.Number,               // follow lists
		nip11.EncryptedDirectMessage.Number,   // encrypted DM
		nip11.MappingNostrKeysToDNS.Number,    // DNS
		nip11.EventDeletion.Number,            // event delete
		nip11.RelayInformationDocument.Number, // relay information document
		nip11.NostrMarketplace.Number,         // marketplace
		nip11.Reposts.Number,                  // reposts
		nip11.Bech32EncodedEntities.Number,    // bech32 encodings
		nip11.LongFormContent.Number,          // long form
		nip11.PublicChat.Number,               // public chat
		nip11.UserStatuses.Number,             // user statuses
		nip11.Authentication.Number,           // auth
		nip11.CountingResults.Number,          // count requests
	)
	db := &badger.Backend{Path: dataDir, Log: nil}
	if err = db.Init(); rl.E.Chk(err) {
		rl.E.F("unable to start database: '%s'", err)
		os.Exit(1)
	}
	switch {
	case args.ImportCmd != nil:
		rl.Import(db, args.ImportCmd.FromFile)
	case args.ExportCmd != nil:
		rl.Export(db, args.ExportCmd.ToFile)
	default:
		rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
		rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
		rl.CountEvents = append(rl.CountEvents, db.CountEvents)
		rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)
		rl.I.Ln("listening on", args.Listen)
		rl.E.Chk(http.ListenAndServe(args.Listen, rl))
	}
}
