package accesscontrol

import (
	"sync"
	"time"
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

// UserID is the required data for an entry in T
type UserID struct {
	// Role is the type of permissions granted to the user
	Role `json:"role"`
	// PubKeyHex is the hex encoded form of the public key related to this role,
	// as a string because events contain a string and this can be quickly
	// checked.
	PubKeyHex string `json:"pubkey"`
	// Expires defines a time when the role must be revised or it ceases to be
	// in force. If a user's expiry is in the past they have no permission
	// anymore.
	Expires time.Time
}

type T struct {
	// Owners can change the Administrators list and can request any event that
	// would otherwise require auth (eg DMs between users) - the reason being
	// that they ultimately can bypass anything built into this software anyway
	// due to "physical" access. All other permissions relate to remote access.
	// The owners cannot be remotely changed via the ACL messages, it must be
	// manually specified in the configuration file.
	Users []*UserID `json:"users,omitempty"`
	// Public means that the relay allows read access to any user without
	// authentication.
	Public bool `json:"public,omitempty"`
	// PublicAuth indicates whether all access should require authentication
	// using NIP-42
	PublicAuth bool `json:"public_auth_required,omitempty"`
	// Access to this data must be mutual exclusive locked
	// so concurrent threads can access and change this data without races.
	sync.Mutex
}

// Closure is a function type that can be used to inspect the user list. This
// is so that concurrent safety can be enforced.
type Closure func(list []*UserID)

func (ac *T) Get(r Role, fn Closure) {
	ac.Mutex.Lock()
	defer ac.Mutex.Unlock()
	var privList []*UserID
	for _, u := range ac.Users {
		if u.Role == r {
			privList = append(privList, u)
		}
	}
	fn(privList)
}

// AuthRequired returns whether authentication is always required
func (ac *T) AuthRequired() bool {
	ac.Mutex.Lock()
	defer ac.Mutex.Unlock()
	return ac.PublicAuth
}

func (ac *T) GetPrivilege(key string) (r Role) {
	ac.Mutex.Lock()
	defer ac.Mutex.Unlock()
	for _, u := range ac.Users {
		if u.PubKeyHex == key {
			r = u.Role
			break
		}
	}
	return
}
