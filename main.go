package main

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/replicatr"
	"mleku.dev/git/nostr/number"
	"mleku.dev/git/nostr/relayinfo"
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

func main() {
	replicatr.Main(os.Args)
}
