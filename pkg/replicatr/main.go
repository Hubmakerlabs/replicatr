package replicatr

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/apputil"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/IC"
	"github.com/alexflint/go-arg"
	"mleku.dev/git/interrupt"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/eventstore"
	"mleku.dev/git/nostr/eventstore/badger"
	"mleku.dev/git/nostr/keys"
	"mleku.dev/git/nostr/number"
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

var nips = number.List{
	relayinfo.BasicProtocol.Number,                  // NIP1 events, envelopes and filters
	relayinfo.FollowList.Number,                     // NIP2 contact list and pet names
	relayinfo.EncryptedDirectMessage.Number,         // NIP4 encrypted DM
	relayinfo.MappingNostrKeysToDNS.Number,          // NIP5 DNS
	relayinfo.EventDeletion.Number,                  // NIP9 event delete
	relayinfo.RelayInformationDocument.Number,       // NIP11 relay information document
	relayinfo.GenericTagQueries.Number,              // NIP12 generic tag queries
	relayinfo.NostrMarketplace.Number,               // NIP15 marketplace
	relayinfo.EventTreatment.Number,                 // NIP16
	relayinfo.Reposts.Number,                        // NIP18 reposts
	relayinfo.Bech32EncodedEntities.Number,          // NIP19 bech32 encodings
	relayinfo.CommandResults.Number,                 // NIP20
	relayinfo.SomethingSomething.Number,             // NIP22
	relayinfo.LongFormContent.Number,                // NIP23 long form
	relayinfo.PublicChat.Number,                     // NIP28 public chat
	relayinfo.ParameterizedReplaceableEvents.Number, // NIP33
	relayinfo.ExpirationTimestamp.Number,            // NIP40
	relayinfo.UserStatuses.Number,                   // NIP38 user statuses
	relayinfo.Authentication.Number,                 // NIP42 auth
	relayinfo.CountingResults.Number,                // NIP45 count requests
}

func Main(osArgs []string, c context.T, cancel context.F) {
	tmp := os.Args
	os.Args = osArgs
	var log, chk = slog.New(os.Stderr)
	arg.MustParse(&args)
	os.Args = tmp
	// set logging level if non-default was set in args
	if args.LogLevel != "info" {
		for i := range slog.LevelSpecs {
			if slog.LevelSpecs[i].Name[:1] == strings.
				ToLower(args.LogLevel[:1]) {

				slog.SetLogLevel(i)
			}
		}
	}
	log.D.S(args)
	runtime.GOMAXPROCS(args.MaxProcs)
	var err error
	var dataDirBase string
	if dataDirBase, err = os.UserHomeDir(); chk.E(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, args.Profile)
	log.D.F("using profile directory: %s", dataDir)
	if !apputil.FileExists(dataDir) {
		args.InitCfgCmd = &app.InitCfg{}
	}
	infoPath := filepath.Join(dataDir, "info.json")
	configPath := filepath.Join(dataDir, "config.json")
	// inf := *relayinfo.NewInfo(&relayinfo.T{Nips: nips})
	var inf relayinfo.T
	// generate a relay identity key if one wasn't given
	if args.SecKey == "" {
		args.SecKey = keys.GeneratePrivateKey()
	}
	// initialize configuration with whatever has been read from the CLI.
	if args.InitCfgCmd != nil {
		if args.Pubkey, err = keys.GetPublicKey(args.SecKey); chk.E(err) {
		}
		inf = relayinfo.T{
			Name:        args.Name,
			Description: args.Description,
			PubKey:      args.Pubkey,
			Contact:     args.Contact,
			Nips:        nips,
			Software:    AppName,
			Version:     Version,
			Limitation: relayinfo.Limits{
				MaxMessageLength: app.MaxMessageSize,
				Oldest:           1640305963,
				AuthRequired:     args.AuthRequired,
				PaymentRequired:  args.AuthRequired,
				RestrictedWrites: args.AuthRequired,
				MaxSubscriptions: 50,
			},
			Retention:      object.T{},
			RelayCountries: tag.T{},
			LanguageTags:   tag.T{},
			Tags:           tag.T{},
			PostingPolicy:  "",
			PaymentsURL:    "https://gfy.mleku.dev",
			Fees: relayinfo.Fees{
				Admission: []relayinfo.Admission{
					{Amount: 100000000, Unit: "satoshi"},
				},
			},
			Icon: args.Icon,
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
		// log.T.S(conf)
		// if fields are empty, overwrite them with the cli args file
		// versions
		if args.Listen != "" {
			conf.Listen = args.Listen
		}
		if args.Profile != "" {
			conf.Profile = args.Profile
		}
		if args.Name != "" {
			conf.Name = args.Name
		}
		if args.Description != "" {
			conf.Description = args.Description
		}
		if args.Pubkey != "" {
			conf.Pubkey = args.Pubkey
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
		if args.DBHighWater != 0 {
			conf.DBHighWater = args.DBHighWater
		}
		if args.GCFrequency != 0 {
			conf.GCFrequency = args.GCFrequency
		}
		if conf.Pubkey, err = keys.GetPublicKey(conf.SecKey); chk.E(err) {
		}
		// log.D.S(conf)
		if err = inf.Load(infoPath); chk.E(err) {
			inf = relayinfo.T{
				Name:        args.Name,
				Description: args.Description,
				PubKey:      args.Pubkey,
				Contact:     args.Contact,
				Nips:        nips,
				Software:    AppName,
				Version:     Version,
				Limitation: relayinfo.Limits{
					MaxMessageLength: app.MaxMessageSize,
					Oldest:           1640305963,
					AuthRequired:     args.AuthRequired,
					PaymentRequired:  args.AuthRequired,
					RestrictedWrites: args.AuthRequired,
					MaxSubscriptions: 50,
				},
				Retention:      object.T{},
				RelayCountries: tag.T{},
				LanguageTags:   tag.T{},
				Tags:           tag.T{},
				PostingPolicy:  "",
				PaymentsURL:    "https://gfy.mleku.dev",
				Fees: relayinfo.Fees{
					Admission: []relayinfo.Admission{
						{21000000, "bitcoin"},
					},
				},
				Icon: args.Icon,
			}
			log.D.F("failed to load relay information document: '%s' "+
				"deriving from config", err)
		}
	}
	if args.AuthRequired {
		conf.AuthRequired = true
		inf.Limitation.AuthRequired = true
	}
	log.D.S(&inf)
	// c, cancel := context.Cancel(context.Bg())
	var wg sync.WaitGroup
	rl := app.NewRelay(c, cancel, &inf, &args)
	var db eventstore.Store
	var parameters []string
	parameters = []string{
		fmt.Sprint(conf.DBSizeLimit),
		fmt.Sprint(conf.DBLowWater),
		fmt.Sprint(conf.DBHighWater),
		fmt.Sprint(conf.GCFrequency),
	}
	badgerDB := &badger.Backend{
		Path: dataDir,
		// Ctx:    c,
		// WG:     &wg,
		// Info:   rl.Info,
		// Params: parameters,
	}
	// if we are wiping we don't want to init db normally
	if args.Wipe != nil {
		conf.DBSizeLimit = 0
	}
	switch rl.Config.EventStore {
	case "ic":
		parameters = append([]string{
			rl.Config.CanisterAddr,
			rl.Config.CanisterID,
		}, parameters...)
		db = &IC.Backend{
			Badger: badgerDB,
			Ctx:    c,
			WG:     &wg,
			Inf:    rl.Info,
			Params: parameters,
		}
	case "badger":
		db = badgerDB
	}

	if err = db.Init(); chk.E(err) {
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

}
