package app

import (
	"errors"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
)

type Role int

// ACL roles
const (
	// RoleOwner is the role of a user who has all privileges except for
	// altering others with the same role.
	RoleOwner Role = iota
	// RoleAdmin is the role that can change all lower roles except for adding
	// and removing administrators.
	RoleAdmin
	// RoleWriter is a user who has the right to add events to the relay.
	RoleWriter
	// RoleReader is a user who may search and retrieve events from the relay.
	RoleReader
	// RoleDenied is a blacklisted user who may not read from or write to the
	// relay.
	RoleDenied
	// RoleNone is the tombstone event that puts the user in the same role as an
	// unauthenticated user (which may mean the same as RoleDenied in effect.
	RoleNone
)

const (
	ACLKind = kind.ACLEvent
)

// RoleStrings are the human readable form of the role enums.
var RoleStrings = []string{
	"Owner",
	"Admin",
	"Writer",
	"Reader",
	"Denied",
	"None",
}

type (
	// ACLEntry is
	ACLEntry struct {
		// EventID is the event ID that creates the ACLEntry.
		EventID eventid.T
		// Role is the role now in force for the pubkey for this ACLEntry.
		Role Role
		// Pubkey is the public key that associates with the Role.
		Pubkey string
		// AuthKey is the public key of the user with RoleAdmin or RoleOwner
		// that requested the change.
		AuthKey string
		// Replaces specifies the event ID (if any) that this entry replaces.
		Replaces eventid.T
		// Created is the created_at field of the event ID of this pubkey being
		// first added to the ACL
		Created int64
		// LastModified is the created at of the most recent event that altered
		// this ACLEntry.
		LastModified int64
		// Expires is the unix timestamp after which this entry is no longer in
		// force and in effect reverts to RoleNone.
		Expires int64
	}
	// ACL is the state information of the relay's Access Control List (ACL).
	ACL struct {
		sync.Mutex
		Entries []*ACLEntry
	}
)

// AddEntry adds or modifies an entry in the ACL.
func (a *ACL) AddEntry(entry *ACLEntry) (err error) {
	if entry == nil {
		return errors.New("nil entry for ACL")
	}
	// set last modified timestamp to now
	entry.LastModified = time.Now().Unix()
	// scan for duplicate and replace if found
	a.Lock()
	defer a.Unlock()
	// if there is an existing entry relating to this pubkey, this new one
	// replaces it.
	for i, v := range a.Entries {
		if v.Pubkey == entry.Pubkey {
			entry.Replaces = v.EventID
			a.Entries[i] = entry
			log.D.Ln("replacing entry for key '%s' role '%s'",
				entry.Pubkey, RoleStrings[entry.Role])
			return
		}
	}
	a.Entries = append(a.Entries, entry)
	return
}

func (a *ACLEntry) ToEvent() (ev event.T) {

	return
}

func ACLFromEvent(ev event.T) (a *ACLEntry) {

	return
}
