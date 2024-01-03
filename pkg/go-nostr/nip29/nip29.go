package nip29

import (
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"golang.org/x/exp/slices"
)

type Role struct {
	Name        string
	Permissions map[Permission]struct{}
}

type Permission = string

const (
	PermAddUser          Permission = "add-user"
	PermEditMetadata     Permission = "edit-metadata"
	PermDeleteEvent      Permission = "delete-event"
	PermRemoveUser       Permission = "remove-user"
	PermAddPermission    Permission = "add-permission"
	PermRemovePermission Permission = "remove-permission"
	PermEditGroupStatus  Permission = "edit-group-status"
)

type KindRange []int

var ModerationEventKinds = KindRange{
	event.KindSimpleGroupAddUser,
	event.KindSimpleGroupRemoveUser,
	event.KindSimpleGroupEditMetadata,
	event.KindSimpleGroupAddPermission,
	event.KindSimpleGroupRemovePermission,
	event.KindSimpleGroupDeleteEvent,
	event.KindSimpleGroupEditGroupStatus,
}

var MetadataEventKinds = KindRange{
	event.KindSimpleGroupMetadata,
	event.KindSimpleGroupAdmins,
	event.KindSimpleGroupMembers,
}

func (kr KindRange) Includes(kind int) bool {
	_, ok := slices.BinarySearch(kr, kind)
	return ok
}
