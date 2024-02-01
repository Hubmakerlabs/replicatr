package replicatr

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"mleku.online/git/ec/secp256k1"
)

const (
	ACLfilename = "acl.json"
)

type Role string

const (
	RoleOwner         Role = "owner"
	RoleAdministrator Role = "administrator"
	RoleWriter        Role = "writer"
	RoleReader        Role = "reader"
)

// UserID is the required data for an entry in AccessControl
type UserID struct {
	Role      `json:"role"`
	pubkey    *secp256k1.PublicKey
	PubKeyHex string `json:"pubkey"`
}

type AccessControl struct {
	// Owners can change the Administrators list and can request any event that
	// would otherwise require auth (eg DMs between users) - the reason being
	// that they ultimately can bypass anything built into this software anyway
	// due to "physical" access. All other permissions relate to remote access.
	// The owners cannot be remotely changed via the ACL messages, it must be
	// manually specified in the configuration file.
	Users []*UserID `json:"users,omitempty"`
	// Access to this data must be mutual exclusive locked so concurrent threads
	// can access and change this data without races.
	mx sync.Mutex
}

type ACLClosure func(list []*UserID)

func (ac *AccessControl) Get(r Role, fn ACLClosure) {
	ac.mx.Lock()
	defer ac.mx.Unlock()
	var privList []*UserID
	for _, u := range ac.Users {
		if u.Role == r {
			privList = append(privList, u)
		}
	}
	fn(privList)
}

func (ac *AccessControl) Enabled() bool {
	ac.mx.Lock()
	defer ac.mx.Unlock()
	return len(ac.Users) > 0
}

func (ac *AccessControl) GetPrivilege(key string) (r Role) {
	ac.mx.Lock()
	defer ac.mx.Unlock()
	for _, u := range ac.Users {
		if u.PubKeyHex == key {
			r = u.Role
			break
		}
	}
	return
}

func (rl *Relay) LoadACL(filename string) (err error) {
	rl.ac.mx.Lock()
	defer rl.ac.mx.Unlock()
	var b []byte
	if b, err = os.ReadFile(filename); log.Fail(err) {
		return
	}
	if err = json.Unmarshal(b, &rl.ac); log.Fail(err) {
		return
	}
	rl.Log.T.F("read ACL config from file '%s'\n%s", filename, string(b))
	var owners int
	rl.ac.Get(RoleOwner, func(list []*UserID) {
		for _, u := range list {
			if u.Role == RoleOwner {
				owners++
			}
		}
	})
	if owners < 1 {
		rl.Log.W.Ln("no owners set in access control file:", filename,
			"remote access to change administrators/readers/writers disabled")
	}
	return
}

func (rl *Relay) SaveACL(filename string) (err error) {
	rl.ac.mx.Lock()
	defer rl.ac.mx.Unlock()
	var b []byte
	if b, err = json.Marshal(&rl.ac); log.Fail(err) {
		return
	}
	rl.Log.T.F("writing ACL config to file '%s'\n%s", filename, string(b))
	if err = os.WriteFile(filename, b, 0700); log.Fail(err) {
		return
	}
	return
}

// FilterAccessControl interacts between filters and the privileges of the
// users according to the access control settings of the relay, checking whether
// the request is authorised, if not, requesting authorisation.
//
// If there is an ACL configured, it acts as a whitelist, no access without
// being on the ACL.
//
// If the message is a private message, only authenticated users may get these
// events who also match one of the parties in the conversation.
func (rl *Relay) FilterAccessControl(c context.T, f *filter.T) (reject bool, msg string) {

	var aclPresent bool
	if rl.ac != nil {
		// if there is no access control roles enabled we don't have to
		// check unless it's a privileged message kind.
		aclPresent = rl.ac.Enabled()
	}
	// check if the request filter kinds are privileged
	var privileged bool
	for i := range f.Kinds {
		privileged = kinds.IsPrivileged(f.Kinds[i])
		if privileged {
			break
		}
	}
	ws := GetConnection(c)
	// if access requires auth, check that auth is present.
	if (privileged || aclPresent) && ws.AuthPubKey == "" {
		// send out authorization request
		RequestAuth(c)
		select {
		case <-ws.Authed:
			rl.D.Ln("user authed")
		case <-c.Done():
			rl.D.Ln("context canceled while waiting for auth")
		case <-time.After(5 * time.Second):
			if ws.AuthPubKey == "" {
				return true, "Authorization timeout"
			}
		}
	}
	// if the user has now authed we can check if they have privileges
	if aclPresent {
		if ws.AuthPubKey == "" {
			// acl enabled but no pubkey, unauthorized
			return true, "Unauthorized"
		}
		r := rl.ac.GetPrivilege(ws.AuthPubKey)
		if r == "" {
			// ACL enabled but user not found in ACL
			return true, "Unauthorized"
		}
		if r == RoleOwner {
			// owners have access to the system, there is no sense in
			// blocking their access. all other roles are delegated and
			// usually meaning not having access to the system the relay
			// runs on.
			return
		}
	}
	if !privileged {
		// no ACL in force and not a privileged message type, accept
		return
	}
	log.D.Ln(ws.AuthPubKey, ws.RealRemote, f.ToObject().String())
	receivers, ok := f.Tags["#p"]
	var parties tag.T
	if ok {
		parties = append(f.Authors, receivers...)
	}
	log.D.Ln("parties", parties)
	switch {
	case ws.AuthPubKey == "":
		// not authenticated
		return true, "restricted: this relay does not serve privileged events" +
			" to unauthenticated users, does your client implement NIP-42?"
	case parties.Contains(ws.AuthPubKey):
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
