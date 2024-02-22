package app

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"mleku.dev/git/ec/schnorr"
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
	ACLKind     = kind.ACLEvent
	ReplacesTag = "replaces"
	ExpiryTag   = "expiry"
)

// RoleStrings are the human readable form of the role enums.
var RoleStrings = []string{
	"owner",
	"admin",
	"writer",
	"reader",
	"denied",
	"none",
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
		Created timestamp.T
		// LastModified is the created at of the most recent event that altered
		// this ACLEntry.
		LastModified timestamp.T
		// Expires is the unix timestamp after which this entry is no longer in
		// force and in effect reverts to RoleNone.
		Expires timestamp.T
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
	entry.LastModified = timestamp.Now()
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

func (a *ACLEntry) ToEvent() (ev *event.T) {
	ev = &event.T{
		CreatedAt: timestamp.Now(),
		Kind:      ACLKind,
		Tags:      tags.T{{"p", a.EventID.String(), RoleStrings[a.Role]}},
	}
	if a.Expires > 0 {
		ev.Tags = append(ev.Tags, tag.T{ExpiryTag, fmt.Sprint(a.Expires)})
	}
	if a.Replaces != "" {
		ev.Tags = append(ev.Tags, tag.T{ReplacesTag, a.Replaces.String()})
	}
	return
}

func ACLFromEvent(ev event.T) (a *ACLEntry, err error) {
	// first populate the fields that are instantly transferable
	a = &ACLEntry{
		EventID:      ev.ID,
		AuthKey:      ev.PubKey,
		LastModified: ev.CreatedAt,
	}
	// Created must be populated by scanning the database for prior versions
	// or if the entry already exists, for now offloading this to upstream.

	// Role requires converting the string back to a number... the strings must
	// be exactly as in the list RoleStrings. Also there must be a role.
	pTags := ev.Tags.GetAll("p")
	if len(pTags) != 1 {
		err = log.E.Err("other than 1 p tag found: %d %v", len(pTags), pTags)
		return
	}
	pTag := pTags[0]
	if len(pTag) > 3 {
		err = log.E.Err("p tag with insufficient fields found: %d %v",
			len(pTag), pTag)
		return
	}
	a.Pubkey = pTag[1]
	if len(a.Pubkey) != schnorr.PubKeyBytesLen*2 {
		err = log.E.Err("public key with wrong length found: %d %v",
			len(a.Pubkey), a.Pubkey)
		return
	}
	if _, err = hex.Dec(a.Pubkey); chk.D(err) {
		return
	}
	var match bool
	for i, v := range RoleStrings {
		if pTag[2] == v {
			a.Role = Role(i)
			match = true
			break
		}
	}
	if !match {
		err = log.E.Err("no match on role string: %v", pTag)
		return
	}
	// Look for the Expires tag.
	expiryTags := ev.Tags.GetAll("expiry")
	if len(expiryTags) != 1 {
		err = log.E.Err("other than 1 expiry tag found: %d %v",
			len(expiryTags), expiryTags)
		return
	} else {
		expiryTag := expiryTags[0]
		if len(expiryTag) < 2 {
			err = log.E.Err("expiry tag with insufficient fields found: %d %v",
				len(expiryTag), expiryTag)
			return
		}
		expiry := expiryTag[1]
		var exp int64
		if exp, err = strconv.ParseInt(expiry, 10, 64); chk.E(err) {
			return
		}
		a.Expires = timestamp.FromUnix(exp)
	}
	// Look for the replaces tag.
	replacesTags := ev.Tags.GetAll("replaces")
	if len(replacesTags) != 1 {
		err = log.E.Err("other than 1 replaces tag found: %d %v",
			len(replacesTags), replacesTags)
		return
	} else {
		replacesTag := replacesTags[0]
		if len(replacesTag) < 2 {
			err = log.E.Err("expiry tag with insufficient fields found: %d %v",
				len(replacesTag), replacesTag)
			return
		}
		a.Replaces = eventid.T(replacesTag[1])
	}
	return
}
