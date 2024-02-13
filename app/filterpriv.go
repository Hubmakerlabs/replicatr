package app

import (
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
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
func (rl *Relay) FilterPrivileged(c context.T, f *filter.T) (reject bool, msg string) {

	authRequired := rl.Info.Limitation.AuthRequired
	// check if the request filter kinds are privileged
	privileged := kinds.IsPrivileged(f.Kinds...)
	ws := GetConnection(c)
	// if access requires auth, check that auth is present.
	if (privileged || authRequired) && ws.AuthPubKey.Load() == "" {
		// send out authorization request
		RequestAuth(c)
		select {
		case <-ws.Authed:
			// log.D.Ln("user authed", GetAuthed(c))
		case <-c.Done():
			log.T.Ln("context canceled while waiting for auth")
		case <-time.After(5 * time.Second):
			if ws.AuthPubKey.Load() == "" {
				return true, "Authorization timeout"
			}
		}
	}
	// if the user has now authed we can check if they have privileges
	if authRequired && ws.AuthPubKey.Load() == "" {
		// acl enabled but no pubkey, unauthorized
		return true, "Unauthorized"
	}
	if !privileged {
		// no ACL in force and not a privileged message type, accept
		return
	}
	receivers, _ := f.Tags["#p"]
	parties := make(tag.T, len(receivers)+len(f.Authors))
	copy(parties[:len(f.Authors)], f.Authors)
	copy(parties[len(f.Authors):], receivers)
	log.T.Ln(ws.RealRemote.Load(), "parties", parties)
	switch {
	case ws.AuthPubKey.Load() == "":
		// not authenticated
		return true, "restricted: this relay does not serve privileged events" +
			" to unauthenticated users, does your client implement NIP-42?"
	case parties.Contains(ws.AuthPubKey.Load()):
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
