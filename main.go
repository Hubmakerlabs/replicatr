package main

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/IC"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
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

var args, conf app.Config

func main() {
	var log, chk = slog.New(os.Stderr)
	arg.MustParse(&args)
	log.T.S(args)
	runtime.GOMAXPROCS(args.MaxProcs)
	var dataDirBase string
	var err error
	if dataDirBase, err = os.UserHomeDir(); log.E.Chk(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, args.Profile)
	log.D.F("using profile directory: %s", args.Profile)
	infoPath := filepath.Join(dataDir, "info.json")
	configPath := filepath.Join(dataDir, "config.json")
	inf := *nip11.NewInfo(nil)
	// initialize configuration with whatever has been read from the CLI.
	if args.InitCfgCmd != nil {
		// generate a relay identity key if one wasn't given
		if args.SecKey == "" {
			args.SecKey = keys.GeneratePrivateKey()
		}
		inf = nip11.Info{
			Name:        args.Name,
			Description: args.Description,
			PubKey:      args.Pubkey,
			Contact:     args.Contact,
			Software:    AppName,
			Version:     Version,
			Limitation: nip11.Limits{
				MaxMessageLength: app.MaxMessageSize,
				Oldest:           1640305963,
				// AuthRequired:     args.AuthRequired,
			},
			Retention:      object.T{},
			RelayCountries: tag.T{},
			LanguageTags:   tag.T{},
			Tags:           tag.T{},
			PostingPolicy:  "",
			PaymentsURL:    "",
			Fees:           nip11.Fees{},
			Icon:           args.Icon,
		}
		if err = args.Save(configPath); chk.E(err) {
			log.E.F("failed to write relay configuration: '%s'", err)
			os.Exit(1)
		}
		if err = inf.Save(infoPath); chk.E(err) {
			log.E.F("failed to write relay information document: '%s'", err)
			os.Exit(1)
		}
	} else {
		if err = conf.Load(configPath); chk.E(err) {
			log.T.F("failed to load relay configuration: '%s'", err)
			os.Exit(1)
		}
		// if fields are empty, overwrite them with the cli args file
		// versions
		if args.Listen != "" {
			args.Listen = conf.Listen
		}
		if args.Profile != "" {
			args.Profile = conf.Profile
		}
		if args.Name != "" {
			args.Name = conf.Name
		}
		if args.Description != "" {
			args.Description = conf.Description
		}
		if args.Pubkey != "" {
			args.Description = conf.Description
		}
		if args.Contact != "" {
			args.Contact = conf.Contact
		}
		if args.Icon == "" {
			args.Icon = conf.Icon
		}
		// CLI args on "separate" items add to the ones in the config
		if len(args.Whitelist) == 0 {
			args.Whitelist = append(args.Whitelist, conf.Whitelist...)
		}
		if len(args.Owners) == 0 {
			args.Owners = append(args.Owners, conf.Owners...)
		}
		if args.SecKey == "" {
			args.SecKey = conf.SecKey
		}
		if err = inf.Load(infoPath); chk.E(err) {
			inf = nip11.Info{
				Name:        args.Name,
				Description: args.Description,
				PubKey:      args.Pubkey,
				Contact:     args.Contact,
				Software:    AppName,
				Version:     Version,
				Limitation: nip11.Limits{
					MaxMessageLength: app.MaxMessageSize,
					Oldest:           1640305963,
					AuthRequired:     args.AuthRequired,
				},
				Retention:      object.T{},
				RelayCountries: tag.T{},
				LanguageTags:   tag.T{},
				Tags:           tag.T{},
				PostingPolicy:  "",
				PaymentsURL:    "",
				Fees:           nip11.Fees{},
				Icon:           args.Icon,
			}
			log.D.F("failed to load relay information document: '%s' "+
				"deriving from config", err)
		}
	}
	log.T.S(&inf)
	rl := app.NewRelay(&inf, &args)
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
	db := &IC.Backend{
		Badger: &badger.Backend{
			Path: dataDir,
		},
	}
	if err = db.Init(rl.Info); chk.E(err) {
		log.E.F("unable to start database: '%s'", err)
		os.Exit(1)
	}
	rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
	rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
	rl.CountEvents = append(rl.CountEvents, db.CountEvents)
	rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)
	// rl.RejectFilter = append(rl.RejectFilter, rl.FilterPrivileged)
	// rl.RejectCountFilter = append(rl.RejectCountFilter, rl.FilterPrivileged)
	rl.StoreEvent = append(rl.StoreEvent, rl.Chat)
	// Load ACL events

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
