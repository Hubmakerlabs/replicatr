package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/IC"
	"github.com/alexflint/go-arg"
	"mleku.dev/git/nostr/eventstore/badger"
	"mleku.dev/git/nostr/keys"
	"mleku.dev/git/nostr/relayinfo"
	"mleku.dev/git/nostr/tag"
	"mleku.dev/git/nostr/wire/object"
	"mleku.dev/git/slog"
)

var (
	AppName = "replicatr"
	Version = "v0.0.1"
)

var args, conf app.Config

func main() {
	var log, chk = slog.New(os.Stderr)
	arg.MustParse(&args)
	log.D.S(args)
	runtime.GOMAXPROCS(args.MaxProcs)
	var dataDirBase string
	var err error
	if dataDirBase, err = os.UserHomeDir(); chk.E(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, args.Profile)
	log.D.F("using profile directory: %s", args.Profile)
	infoPath := filepath.Join(dataDir, "info.json")
	configPath := filepath.Join(dataDir, "config.json")
	inf := *relayinfo.NewInfo(nil)
	// initialize configuration with whatever has been read from the CLI.
	if args.InitCfgCmd != nil {
		// generate a relay identity key if one wasn't given
		if args.SecKey == "" {
			args.SecKey = keys.GeneratePrivateKey()
		}
		inf = relayinfo.T{
			Name:        args.Name,
			Description: args.Description,
			PubKey:      args.Pubkey,
			Contact:     args.Contact,
			Software:    AppName,
			Version:     Version,
			Limitation: relayinfo.Limits{
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
			Fees:           relayinfo.Fees{},
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
			log.D.F("failed to load relay configuration: '%s'", err)
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
		if args.AuthRequired {
			conf.AuthRequired = true
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
		log.D.S(conf)
		if err = inf.Load(infoPath); chk.E(err) {
			inf = relayinfo.T{
				Name:        args.Name,
				Description: args.Description,
				PubKey:      args.Pubkey,
				Contact:     args.Contact,
				Software:    AppName,
				Version:     Version,
				Limitation: relayinfo.Limits{
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
				Fees:           relayinfo.Fees{},
				Icon:           args.Icon,
			}
			log.D.F("failed to load relay information document: '%s' "+
				"deriving from config", err)
		}
		if args.AuthRequired {
			inf.Limitation.AuthRequired = true
		}
	}
	log.D.S(&inf)
	rl := app.NewRelay(&inf, &args)
	rl.Info.AddNIPs(
		relayinfo.BasicProtocol.Number,            // events, envelopes and filters
		relayinfo.FollowList.Number,               // follow lists
		relayinfo.EncryptedDirectMessage.Number,   // encrypted DM
		relayinfo.MappingNostrKeysToDNS.Number,    // DNS
		relayinfo.EventDeletion.Number,            // event delete
		relayinfo.RelayInformationDocument.Number, // relay information document
		relayinfo.NostrMarketplace.Number,         // marketplace
		relayinfo.Reposts.Number,                  // reposts
		relayinfo.Bech32EncodedEntities.Number,    // bech32 encodings
		relayinfo.LongFormContent.Number,          // long form
		relayinfo.PublicChat.Number,               // public chat
		relayinfo.UserStatuses.Number,             // user statuses
		relayinfo.Authentication.Number,           // auth
		relayinfo.CountingResults.Number,          // count requests
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
	rl.StoreEvent = append(rl.StoreEvent, rl.Chat)
	rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
	rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
	rl.CountEvents = append(rl.CountEvents, db.CountEvents)
	rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)
	rl.OnConnect = append(rl.OnConnect, rl.AuthCheck)
	rl.RejectFilter = append(rl.RejectFilter, rl.FilterPrivileged)
	rl.RejectCountFilter = append(rl.RejectCountFilter, rl.FilterPrivileged)
	rl.OverrideDeletion = append(rl.OverrideDeletion, rl.OverrideDelete)
	// run the chat ACL initialization
	rl.Init()
	srvr := http.Server{
		Addr:    args.Listen,
		Handler: rl,
	}
	go func() {
		log.I.Ln("type 1-7 <enter> to change log levels, type q <enter> to quit")
		var b = make([]byte, 1)
		for {
			_, err = os.Stdin.Read(b)
			if !chk.E(err) {
				switch b[0] {
				case '1':
					fmt.Println("logging off")
					slog.SetLogLevel(slog.Off)
				case '2':
					fmt.Println("logging fatal")
					slog.SetLogLevel(slog.Fatal)
				case '3':
					fmt.Println("logging error")
					slog.SetLogLevel(slog.Error)
				case '4':
					fmt.Println("logging warn")
					slog.SetLogLevel(slog.Warn)
				case '5':
					fmt.Println("logging info")
					slog.SetLogLevel(slog.Info)
				case '6':
					fmt.Println("logging debug")
					slog.SetLogLevel(slog.Debug)
				case '7':
					fmt.Println("logging trace")
					slog.SetLogLevel(slog.Trace)
				case 'q':
					chk.E(srvr.Close())
				}
			}
		}
	}()
	switch {
	case args.ImportCmd != nil:
		rl.Import(db.Badger, args.ImportCmd.FromFile)
	case args.ExportCmd != nil:
		rl.Export(db.Badger, args.ExportCmd.ToFile)
	default:
		log.I.Ln("listening on", args.Listen)
		chk.E(srvr.ListenAndServe())
	}
}
