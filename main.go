package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/IC"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip11"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/object"
	"github.com/alexflint/go-arg"
	"mleku.online/git/slog"
)

var (
	AppName = "replicatr"
	Version = "v0.0.1"
)

var args app.Config

func main() {
	arg.MustParse(&args)
	var dataDirBase string
	var err error
	var log, chk = slog.New(os.Stderr)
	if dataDirBase, err = os.UserHomeDir(); log.E.Chk(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, args.Profile)
	log.D.F("using profile directory: %s", args.Profile)
	rl := app.NewRelay(&nip11.Info{
		Name:        args.Name,
		Description: args.Description,
		PubKey:      args.Pubkey,
		Contact:     args.Contact,
		Software:    AppName,
		Version:     Version,
		Limitation: nip11.Limits{
			MaxMessageLength: app.MaxMessageSize,
			Oldest:           1640305963,
		},
		Retention:      object.T{},
		RelayCountries: tag.T{},
		LanguageTags:   tag.T{},
		Tags:           tag.T{},
		PostingPolicy:  "",
		PaymentsURL:    "",
		Fees:           nip11.Fees{},
		Icon:           args.Icon,
	}, args.Whitelist)
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
	if args.InitCfgCmd != nil {
		// initialize configuration with whatever has been read from the CLI.
		// include adding nip-11 configuration documents to this...
		if err = args.Save(filepath.Join(dataDir, "config.json")); chk.E(err) {
		}
		if err = rl.Info.Save(filepath.Join(dataDir, "config.json")); chk.E(err) {
		}

	}
	db := &IC.Backend{
		Badger: &badger.Backend{
			Path:  dataDir,
			Log:   log,
			Check: chk,
		},
	}
	if err = db.Init(); chk.E(err) {
		log.E.F("unable to start database: '%s'", err)
		os.Exit(1)
	}
	rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
	rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
	rl.CountEvents = append(rl.CountEvents, db.CountEvents)
	rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)
	rl.RejectFilter = append(rl.RejectFilter, rl.FilterPrivileged)
	rl.RejectCountFilter = append(rl.RejectCountFilter, rl.FilterPrivileged)
	switch {
	case args.ImportCmd != nil:
		rl.Import(db.Badger, args.ImportCmd.FromFile)
	case args.ExportCmd != nil:
		rl.Export(db.Badger, args.ExportCmd.ToFile)
	default:
		log.I.Ln("listening on", args.Listen)
		chk.E(http.ListenAndServe(args.Listen, rl))
	}
}
