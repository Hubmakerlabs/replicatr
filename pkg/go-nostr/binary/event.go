package binary

import (
	"encoding/hex"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
)

type Event struct {
	PubKey    [32]byte
	Sig       [64]byte
	ID        [32]byte
	Kind      uint16
	CreatedAt timestamp.T
	Content   string
	Tags      tags.Tags
}

func BinaryEvent(evt *event.T) *Event {
	bevt := Event{
		Tags:      evt.Tags,
		Content:   evt.Content,
		Kind:      uint16(evt.Kind),
		CreatedAt: evt.CreatedAt,
	}

	hex.Decode(bevt.ID[:], []byte(evt.ID))
	hex.Decode(bevt.PubKey[:], []byte(evt.PubKey))
	hex.Decode(bevt.Sig[:], []byte(evt.Sig))

	return &bevt
}

func (bevt *Event) ToNormalEvent() *event.T {
	return &event.T{
		Tags:      bevt.Tags,
		Content:   bevt.Content,
		Kind:      int(bevt.Kind),
		CreatedAt: bevt.CreatedAt,
		ID:        hex.EncodeToString(bevt.ID[:]),
		PubKey:    hex.EncodeToString(bevt.PubKey[:]),
		Sig:       hex.EncodeToString(bevt.Sig[:]),
	}
}
