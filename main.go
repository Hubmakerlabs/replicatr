package main

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/eventstore"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/apputil"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/IC"
	"github.com/alexflint/go-arg"
	"mleku.dev/git/interrupt"
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
	err := arg.Parse(&args)
	if err != nil {
		log.E.Ln(err)
		os.Exit(1)
	}
	// log.D.S(args)
	runtime.GOMAXPROCS(args.MaxProcs)
	var dataDirBase string
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
		apputil.EnsureDir(configPath)
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
		log.T.S(conf)
		// if fields are empty, overwrite them with the cli args file
		// versions
		if args.Listen != "" {
			conf.Listen = args.Listen
		}
		if args.Profile != "" {
			conf.Profile = args.Profile
		}
		if args.AuthRequired {
			conf.AuthRequired = true
			inf.Limitation.AuthRequired = true
		}
		if args.Name != "" {
			conf.Name = args.Name
		}
		if args.Description != "" {
			conf.Description = args.Description
		}
		if args.Pubkey != "" {
			conf.Description = args.Description
		}
		if args.Contact != "" {
			conf.Contact = args.Contact
		}
		if args.Icon == "" {
			conf.Icon = args.Icon
		}
		// CLI args on "separate" items add to the ones in the config
		if len(args.Whitelist) == 0 {
			conf.Whitelist = append(conf.Whitelist, args.Whitelist...)
		}
		if len(args.Owners) == 0 {
			conf.Owners = append(conf.Owners, args.Owners...)
		}
		if args.SecKey == "" {
			conf.SecKey = args.SecKey
		}
		log.I.Ln(args.DBSizeLimit)
		if args.DBSizeLimit != 0 {
			conf.DBSizeLimit = args.DBSizeLimit
		}
		if args.DBLowWater != 0 {
			conf.DBLowWater = args.DBLowWater
		}
		if args.GCFrequency != 0 {
			conf.GCFrequency = args.GCFrequency
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
		inf.Limitation.AuthRequired = args.AuthRequired
	}
	// log.D.S(&inf)
	c, cancel := context.Cancel(context.Bg())
	var wg sync.WaitGroup
	rl := app.NewRelay(c, cancel, &inf, &args)
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
	var db eventstore.Store
	badgerDB := &badger.Backend{
		Path: dataDir,
	}
	var parameters []string
	parameters = []string{
		fmt.Sprint(conf.DBSizeLimit),
		fmt.Sprint(conf.DBLowWater),
		fmt.Sprint(conf.DBHighWater),
		fmt.Sprint(conf.GCFrequency),
	}
	switch rl.Config.EventStore {
	case "ic":
		db = &IC.Backend{
			Ctx:    c,
			Badger: badgerDB,
		}
		parameters = append([]string{
			rl.Config.CanisterAddr,
			rl.Config.CanisterID,
		}, parameters...)
	case "badger":
		db = badgerDB
	}
	if err = db.Init(c, &wg, rl.Info, parameters...); chk.E(err) {
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
	// commenting out the override so GC will work
	// rl.OverrideDeletion = append(rl.OverrideDeletion, rl.OverrideDelete)
	// run the chat ACL initialization
	rl.Init()
	serv := http.Server{
		Addr:    args.Listen,
		Handler: rl,
	}
	interrupt.AddHandler(func() {
		cancel()
		wg.Done()
	})
	go func() {
		for {
			select {
			case <-rl.Ctx.Done():
				chk.E(serv.Close())
				return
			default:
			}
			wg.Wait()
			log.I.Ln("relay now cleanly shut down")
		}
	}()
	wg.Add(1)
	switch {
	case args.Wipe != nil:
		log.D.Ln("wiping database")
		chk.E(rl.Wipe(badgerDB))
		os.Exit(0)
	case args.ImportCmd != nil:
		if rl.Config.EventStore == "badger" {
			rl.Import(badgerDB, args.ImportCmd.FromFile)
		}
	case args.ExportCmd != nil:
		rl.Export(badgerDB, args.ExportCmd.ToFile)
	default:
		log.I.Ln("listening on", args.Listen)
		chk.E(serv.ListenAndServe())
	}

	//serialize context and badger for testing purposes
	contextPath := "./cmd/digestr/app/context.gob"
	file, err := os.Create(contextPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		panic(err)
	}

	badgerPath := "./cmd/digestr/app/badger.gob"

	file, err = os.Create(badgerPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	encoder = gob.NewEncoder(file)
	if err := encoder.Encode(badgerDB); err != nil {
		panic(err)
	}

}
