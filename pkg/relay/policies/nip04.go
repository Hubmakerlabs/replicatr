package policies

import (
	"context"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/relay"
	"golang.org/x/exp/slices"
)

// RejectKind04Snoopers prevents reading NIP-04 messages from people not
// involved in the conversation.
func RejectKind04Snoopers(ctx context.Context, f filter.T) (bool, string) {
	// prevent kind-4 events from being returned to unauthed users,
	//   only when authentication is a thing
	if !slices.Contains(f.Kinds, 4) {
		return false, ""
	}

	ws := relay.GetConnection(ctx)
	senders := f.Authors
	receivers, _ := f.Tags["p"]
	switch {
	case ws.AuthedPublicKey == "":
		// not authenticated
		return true, "restricted: this relay does not serve kind-4 to" +
			" unauthenticated users, does your client implement NIP-42?"
	case len(senders) == 1 && len(receivers) < 2 &&
		(senders[0] == ws.AuthedPublicKey):
		// allowed filter: ws.authed is sole sender (filter specifies one or all
		// receivers)
		return false, ""
	case len(receivers) == 1 && len(senders) < 2 &&
		(receivers[0] == ws.AuthedPublicKey):
		// allowed filter: ws.authed is sole receiver (filter specifies one or
		// all senders)
		return false, ""
	default:
		// restricted filter: do not return any events, even if other elements
		// in filters array were not restricted). client should know better.
		return true, "restricted: authenticated user does not have " +
			"authorization for requested filters."
	}
}
