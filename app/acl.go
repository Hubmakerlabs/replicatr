package app

import (
	"errors"
	"time"
)

type Role int

// ACL roles
const (
	RoleOwner Role = iota
	RoleAdmin
	RoleWriter
	RoleReader
	RoleDenied
)

var RoleStrings = []string{
	"Owner",
	"Admin",
	"Writer",
	"Reader",
	"Denied",
}

type (
	ACLEntry struct {
		Role         Role
		Pubkey       string
		Created      int64
		LastModified int64
		Expires      int64
	}
	ACL struct {
		Entries []*ACLEntry
	}
)

func (a *ACL) AddEntry(entry *ACLEntry) (err error) {
	if entry == nil {
		return errors.New("nil entry for ACL")
	}
	// set last modified timestamp to now
	entry.LastModified = time.Now().Unix()
	// scan for duplicate and replace if found
	for i := range a.Entries {
		if a.Entries[i].Pubkey == entry.Pubkey {
			a.Entries[i] = entry
			log.D.Ln("replacing entry for key '%s' role '%s'",
				entry.Pubkey, RoleStrings[entry.Role])
			return
		}
	}
	a.Entries = append(a.Entries, entry)
	return
}
