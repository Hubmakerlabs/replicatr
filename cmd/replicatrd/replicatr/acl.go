package replicatr

import (
	"encoding/json"
	"os"
	"sync"
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
	PubKeyHex string `json:"pubkey"`
}

type AccessControl struct {
	// Owners can change the Administrators list and can request any event that
	// would otherwise require auth (eg DMs between users) - the reason being
	// that they ultimately can bypass anything built into this software anyway
	// due to "physical" access. All other permissions relate to remote access.
	// The owners cannot be remotely changed via the ACL messages, it must be
	// manually specified in the configuration file.
	Users  []*UserID `json:"users,omitempty"`
	Public bool      `json:"public,omitempty"`
	// PublicAuth indicates whether all access should require authentication
	// using NIP-42 - this
	PublicAuth bool `json:"public_auth_required,omitempty"`
	// Access to this data must be mutual exclusive locked
	// so concurrent threads can access and change this data without races.
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
	if b, err = os.ReadFile(filename); rl.Fail(err) {
		return
	}
	if err = json.Unmarshal(b, &rl.ac); rl.Fail(err) {
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
	if b, err = json.Marshal(&rl.ac); rl.Fail(err) {
		return
	}
	rl.T.F("writing ACL config to file '%s'\n%s", filename, string(b))
	if err = os.WriteFile(filename, b, 0700); rl.Fail(err) {
		return
	}
	return
}
