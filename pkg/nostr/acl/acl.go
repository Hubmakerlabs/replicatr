package acl

import (
	"fmt"
	"os"
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
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Role int

// ACL roles
const (
	// Owner is the role of a user who has all privileges except for
	// altering others with the same role.
	Owner Role = iota
	// Admin is the role that can change all lower roles except for adding
	// and removing administrators.
	Admin
	// Writer is a user who has the right to add events to the relay.
	Writer
	// Reader is a user who may search and retrieve events from the relay.
	Reader
	// Denied is a blacklisted user who may not read from or write to the
	// relay.
	Denied
	// None is the tombstone event that puts the user in the same role as an
	// unauthenticated user (which may mean the same as Denied in effect).
	None
)

const (
	Kind        = kind.ACLEvent
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
	// Entry is
	Entry struct {
		// EventID is the event ID that creates the Entry.
		EventID eventid.T
		// Role is the role now in force for the pubkey for this Entry.
		Role Role
		// Pubkey is the public key that associates with the Role.
		Pubkey string
		// AuthKey is the public key of the user with Admin or Owner
		// that requested the change.
		AuthKey string
		// Replaces specifies the event ID (if any) that this entry replaces.
		Replaces eventid.T
		// Created is the created_at field of the event ID of this pubkey being
		// first added to the ACL
		Created timestamp.T
		// LastModified is the created at of the most recent event that altered
		// this Entry.
		LastModified timestamp.T
		// Expires is the unix timestamp after which this entry is no longer in
		// force and in effect reverts to None.
		Expires timestamp.T
	}
	// T is the state information of the relay's Access Control List (ACL).
	T struct {
		sync.Mutex
		entries []*Entry
	}
)

// AddEntry adds or modifies an entry in the acl.T.
func (ae *T) AddEntry(entry *Entry) (err error) {
	if entry == nil {
		return log.E.Err("nil entry for ACL")
	}
	// set last modified timestamp to now
	entry.LastModified = timestamp.Now()
	// scan for duplicate and replace if found
	ae.Lock()
	defer ae.Unlock()
	// if there is an existing entry relating to this pubkey, this new one
	// replaces it.
	for i, v := range ae.entries {
		if v.Pubkey == entry.Pubkey {
			if v.Role == Owner {
				return log.E.Err("owner entries cannot be modified, only " +
					"possible to change in configuration")
			}
			entry.Replaces = v.EventID
			ae.entries[i] = entry
			log.D.Ln("replacing entry for key '%s' role '%s'",
				entry.Pubkey, RoleStrings[entry.Role])
			return
		}
	}
	ae.entries = append(ae.entries, entry)
	return
}

// DeleteEntry removes a record from the acl.T.
//
// It is not possible to modify or delete an entry with the Owner role.
//
// This will generally be run in response to an event that reverts a user role
// to None, to contain the size of the database as the number of formerly
// privileged users grows in the database. Old records that exceed storage
// limits can be later garbage collected and the events removed eliminating the
// record from the initial process of populating the acl.T from Kind events.
func (ae *T) DeleteEntry(pub string) (err error) {
	if e := ae.Find(pub); e != nil {
		if e.Role == Owner {
			return log.E.Err("owner roles are not modifiable")
		}
		ae.Lock()
		defer ae.Unlock()
		var counter int
		// The most efficient way to remove the entry is to iterate it and keep
		// a second counter that skips the deleted entry and copies all other
		// entries in order. when the item is found, the counter is not
		// increased, but the iteration continues so all entries are shifted
		// back one after the delete point. this is much the same as what would
		// be done if an append operation is done with the before and after
		// segments but simplifies the API for find by not needing an index. The
		// difference being that this does not require optimization by the
		// compiler.
		for _, v := range ae.entries {
			if v.Pubkey != pub {
				ae.entries[counter] = v
				counter++
			}
		}
		// prune off the last entry, which will now be the same as the second
		// last.
		ae.entries = ae.entries[:len(ae.entries)]
	} else {
		return log.D.Err("cannot delete: pubkey not found %s", pub)
	}
	return
}

// Find an Entry in the acl.T that has the matching public key.
func (ae *T) Find(pub string) (e *Entry) {
	ae.Lock()
	defer ae.Unlock()
	for _, v := range ae.entries {
		if v.Pubkey == pub {
			return v
		}
	}
	return
}

// ToEvent converts an Entry into a raw ACL event.T.
//
// note that these are always generated by the ACL configuration interface in
// the relay, after first finding any existing entry to replace.
//
// The ACL control will generate the entry after scanning the existing acl.T and
// then this event will be saved in the database after processing it through
// FromEvent.
func (a *Entry) ToEvent() (ev *event.T) {
	ev = &event.T{
		CreatedAt: timestamp.Now(),
		Kind:      Kind,
		Tags:      tags.T{{"p", a.Pubkey, RoleStrings[a.Role]}},
	}
	if a.Expires > 0 {
		ev.Tags = append(ev.Tags, tag.T{ExpiryTag, fmt.Sprint(a.Expires)})
	}
	if a.Replaces != "" {
		ev.Tags = append(ev.Tags, tag.T{ReplacesTag, a.Replaces.String()})
	}
	return
}

// FromEvent processes an event.T and imports it into the acl.T.
//
// The ACL control system will in fact generate an Entry first, run
// Entry.ToEvent to derive a properly formatted event, sign it, and then run
// FromEvent to validate it after which it will then sign it and store the event
// into the database so it is available for searches and for initializing the
// acl.T at startup to configure the ACL.
func (ae *T) FromEvent(ev *event.T) (e *Entry, err error) {
	// first populate the fields that are instantly transferable
	e = &Entry{
		EventID:      ev.ID,
		AuthKey:      ev.PubKey,
		LastModified: ev.CreatedAt,
	}
	// If the pubkey appears already in the in-memory ACL copy in its Created
	// timestamp to maintain the record's provenance efficiently.
	previous := ae.Find(ev.PubKey)
	if previous != nil {
		e.Created = previous.Created
	}
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
	e.Pubkey = pTag[1]
	if len(e.Pubkey) != schnorr.PubKeyBytesLen*2 {
		err = log.E.Err("public key with wrong length found: %d %v",
			len(e.Pubkey), e.Pubkey)
		return
	}
	if _, err = hex.Dec(e.Pubkey); chk.D(err) {
		return
	}
	var match bool
	for i, v := range RoleStrings {
		if pTag[2] == v {
			e.Role = Role(i)
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
		e.Expires = timestamp.FromUnix(exp)
	}
	// Look for the replaces tag.
	replacesTags := ev.Tags.GetAll("replaces")
	if len(replacesTags) > 1 {
		err = log.E.Err("other than 1 replaces tag found: %d %v",
			len(replacesTags), replacesTags)
		return
	} else if len(replacesTags) > 0 {
		replacesTag := replacesTags[0]
		if len(replacesTag) < 2 {
			err = log.E.Err("expiry tag with insufficient fields found: %d %v",
				len(replacesTag), replacesTag)
			return
		}
		// this event ID should match the one in the current acl.T
		replaces := replacesTag[1]
		if previous != nil {
			if replaces != previous.EventID.String() {
				// this shouldn't happen because that entry should be the latest
				// and this event is relay-internal. Log this for forensics.
				log.W.Ln("replaces field in event does not match the latest" +
					" in the current ACL")
			}
		}
		e.Replaces = eventid.T(replaces)
	}
	if err = ae.AddEntry(e); chk.E(err) {
		return
	}
	return
}
