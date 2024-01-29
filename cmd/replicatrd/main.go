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

var args struct {
	Listen      string `arg:"-l,--listen" default:"0.0.0.0:3334"`
	Profile     string `arg:"-p,--profile" default:"replicatr"`
	Name        string `arg:"-n,--name" default:"replicatr relay"`
	Description string `arg:"--description"`
	Pubkey      string `arg:"-k,--pubkey"`
	Contact     string `arg:"-c,--contact"`
	Icon        string `arg:"-i,--icon" default:"https://i.nostr.build/n8vM.png"`
}

var (
	AppName = "replicatr"
	Version = "v0.0.1"
)

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
	db := &badger.BadgerBackend{Path: dataDir, Log: log}
	if err = db.Init(); rl.E.Chk(err) {
		rl.E.F("unable to start database: '%s'", err)
		os.Exit(1)
	}
	rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
	rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
	rl.CountEvents = append(rl.CountEvents, db.CountEvents)
	rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)
	rl.I.Ln("listening on", args.Listen)
	rl.E.Chk(http.ListenAndServe(args.Listen, rl))
}
