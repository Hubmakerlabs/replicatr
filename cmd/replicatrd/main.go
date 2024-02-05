package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/Hubmakerlabs/replicatr/cmd/replicatrd/replicatr"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/IC"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip11"
	"github.com/alexflint/go-arg"
	"mleku.online/git/slog"
)

var (
	AppName = "replicatr"
	Version = "v0.0.1"
)

var args replicatr.Config

func main() {
	arg.MustParse(&args)
	var dataDirBase string
	var err error
	var log = slog.New(os.Stderr)
	if dataDirBase, err = os.UserHomeDir(); log.E.Chk(err) {
		os.Exit(1)
	}
	dataDir := filepath.Join(dataDirBase, args.Profile)
	log.D.F("using profile directory: '%s", args.Profile)
	var ac *replicatr.AccessControl
	if args.InitCfgCmd != nil {
		// initialize configuration with whatever has been read from the CLI.
		// include adding nip-11 configuration documents to this...
	}
	rl := replicatr.NewRelay(log, &nip11.Info{
		Name:        args.Name,
		Description: args.Description,
		PubKey:      args.Pubkey,
		Contact:     args.Contact,
		Software:    AppName,
		Version:     Version,
		Limitation: &nip11.Limits{
			MaxMessageLength: replicatr.MaxMessageSize,
			Oldest:           1640305963,
		},
		RelayCountries: nil,
		LanguageTags:   nil,
		Tags:           nil,
		PostingPolicy:  "",
		PaymentsURL:    "",
		Fees:           &nip11.Fees{},
		Icon:           args.Icon,
	}, args.Whitelist, ac)
	aclPath := filepath.Join(dataDir, replicatr.ACLfilename)
	// initialise ACL if command is called. Note this will overwrite an existing
	// configuration.
	if args.InitACLCmd != nil {
		if !keys.IsValid32ByteHex(args.InitACLCmd.Owner) {
			log.E.Ln("invalid owner public key")
			os.Exit(1)
		}
		rl.AccessControl = &replicatr.AccessControl{
			Users: []*replicatr.UserID{
				{
					Role:      replicatr.RoleOwner,
					PubKeyHex: args.InitACLCmd.Owner,
				},
			},
			Public:     args.InitACLCmd.Public,
			PublicAuth: args.InitACLCmd.Auth,
		}
		rl.Info.Limitation.AuthRequired = args.InitACLCmd.Auth
		rl.I.Ln("auth required")
		// if the public flag is set, add an empty reader to signal public reader
		if err = rl.SaveACL(aclPath); rl.E.Chk(err) {
			panic(err)
			// this is probably a fatal error really
		}
		log.I.Ln("access control base configuration saved and ready to use")
	}
	// load access control list
	if err = rl.LoadACL(aclPath); rl.W.Chk(err) {
		rl.W.Ln("no access control configured for relay")
	}
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
			Log:  slog.New(os.Stderr),
		},
	}
	if err = db.Init(); rl.E.Chk(err) {
		rl.E.F("unable to start database: '%s'", err)
		os.Exit(1)
	}
	rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
	rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
	rl.CountEvents = append(rl.CountEvents, db.CountEvents)
	rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)
	rl.RejectFilter = append(rl.RejectFilter, rl.FilterAccessControl)
	rl.RejectCountFilter = append(rl.RejectCountFilter, rl.FilterAccessControl)
	switch {
	case args.ImportCmd != nil:
		rl.Import(db.Badger, args.ImportCmd.FromFile)
	case args.ExportCmd != nil:
		rl.Export(db.Badger, args.ExportCmd.ToFile)
	default:
		rl.I.Ln("listening on", args.Listen)
		rl.E.Chk(http.ListenAndServe(args.Listen, rl))
	}
}
