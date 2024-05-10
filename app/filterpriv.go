package app

import (
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/auth"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/closedenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
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
	privileged := kinds.IsPrivileged(f.Kinds...)
	// if access requires auth, check that auth is present.
	if (privileged && authRequired) && ws.AuthPubKey() == "" {
		var reason string
		if privileged {
			reason = "this relay only sends privileged events to parties to the event"
		} else if authRequired {
			reason = "this relay requires authentication for all access"
		}
		log.I.Ln(reason)
		chk.E(ws.WriteEnvelope(&closedenvelope.T{
			ID:     id,
			Reason: normalize.Reason(reason, auth.Required),
		}))
		// send out authorization request
		RequestAuth(c, "REQ")
	out:
		for {
			select {
			case <-ws.Authed:
				log.I.Ln("user authed", ws.RealRemote(), ws.AuthPubKey())
				break out
			case <-c.Done():
				log.D.Ln("context canceled while waiting for auth")
				break out
			case <-time.After(5 * time.Second):
				if ws.AuthPubKey() == "" {
					return true,
						log.I.Err("Authorization timeout from ",
							ws.RealRemote()).Error()
				}
			}
		}
	}
	// if the user has now authed we can check if they have privileges
	if authRequired && ws.AuthPubKey() == "" {
		// acl enabled but no pubkey, unauthorized
		log.I.Ln("waited for auth but none came")
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
