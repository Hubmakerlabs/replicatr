package app

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

// FilterPrivileged interacts between filters and the privileges of the
// users according to the access control settings of the relay, checking whether
// the request is authorised, if not, requesting authorisation.
//
// If there is an ACL configured, it acts as a whitelist, no access without
// being on the ACL.
//
// If the message is a private message, only authenticated users may get these
// events who also match one of the parties in the conversation.
func (rl *Relay) FilterPrivileged(c context.T, id subscriptionid.T,
	f *filter.T) (reject bool, msg string) {

	ws := GetConnection(c)
	authRequired := rl.Info.Limitation.AuthRequired
	if !authRequired {
		return
	}
	var allow bool
	for _, v := range rl.Config.AllowIPs {
		if ws.RealRemote() == v {
			allow = true
			break
		}
	}
	if allow {
		return
	}
	// check if the request filter kinds are privileged
	if !kinds.IsPrivileged(f.Kinds...) {
		return
	}
	if !rl.IsAuthed(c, "privileged") {
		return
	}
	receivers, _ := f.Tags["#p"]
	parties := make(tag.T, len(receivers)+len(f.Authors))
	copy(parties[:len(f.Authors)], f.Authors)
	copy(parties[len(f.Authors):], receivers)
	log.D.Ln(ws.RealRemote(), "parties", parties, "querant", ws.AuthPubKey())
	switch {
	case ws.AuthPubKey() == "":
		// not authenticated
		return true, "restricted: this relay does not serve privileged events" +
			" to unauthenticated users, does your client implement NIP-42?"
	case parties.Contains(ws.AuthPubKey()):
		// if the authed key is party to the messages, either as authors or
		// recipients then they are permitted to see the message.
		return
	default:
		// restricted filter: do not return any events, even if other elements
		// in filters array were not restricted). client should know better.
		return true, "restricted: authenticated user does not match either " +
			"party in privileged message type"
	}
}
