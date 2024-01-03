package nip29

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
)

type Group struct {
	ID      string
	Name    string
	Picture string
	About   string
	Members map[string]*Role
	Private bool
	Closed  bool

	LastMetadataUpdate timestamp.Timestamp
}

func (group Group) ToMetadataEvent() *event.T {
	evt := &event.T{
		Kind:      39000,
		CreatedAt: group.LastMetadataUpdate,
		Content:   group.About,
		Tags: tags.Tags{
			tags.Tag{"d", group.ID},
		},
	}
	if group.Name != "" {
		evt.Tags = append(evt.Tags, tags.Tag{"name", group.Name})
	}
	if group.About != "" {
		evt.Tags = append(evt.Tags, tags.Tag{"about", group.Name})
	}
	if group.Picture != "" {
		evt.Tags = append(evt.Tags, tags.Tag{"picture", group.Picture})
	}

	// status
	if group.Private {
		evt.Tags = append(evt.Tags, tags.Tag{"private"})
	} else {
		evt.Tags = append(evt.Tags, tags.Tag{"public"})
	}
	if group.Closed {
		evt.Tags = append(evt.Tags, tags.Tag{"closed"})
	} else {
		evt.Tags = append(evt.Tags, tags.Tag{"open"})
	}

	return evt
}

func (group *Group) MergeInMetadataEvent(evt *event.T) error {
	if evt.Kind != event.KindSimpleGroupMetadata {
		return fmt.Errorf("expected kind %d, got %d", event.KindSimpleGroupMetadata, evt.Kind)
	}

	if evt.CreatedAt <= group.LastMetadataUpdate {
		return fmt.Errorf("event is older than our last update (%d vs %d)", evt.CreatedAt, group.LastMetadataUpdate)
	}

	group.LastMetadataUpdate = evt.CreatedAt
	group.ID = evt.Tags.GetD()
	group.Name = group.ID

	if tag := evt.Tags.GetFirst([]string{"name", ""}); tag != nil {
		group.Name = (*tag)[1]
	}
	if tag := evt.Tags.GetFirst([]string{"about", ""}); tag != nil {
		group.About = (*tag)[1]
	}
	if tag := evt.Tags.GetFirst([]string{"picture", ""}); tag != nil {
		group.Picture = (*tag)[1]
	}

	if tag := evt.Tags.GetFirst([]string{"private"}); tag != nil {
		group.Private = true
	}
	if tag := evt.Tags.GetFirst([]string{"closed"}); tag != nil {
		group.Closed = true
	}

	return nil
}
