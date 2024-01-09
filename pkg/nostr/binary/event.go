package binary

import (
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

var log = log2.GetStd()

type Event struct {
	PubKey    [32]byte
	Sig       [64]byte
	ID        [32]byte
	Kind      uint16
	CreatedAt timestamp.T
	Content   string
	Tags      tags.T
}

func BinaryEvent(evt *event.T) (be *Event) {
	be = &Event{
		Tags:      evt.Tags,
		Content:   evt.Content,
		Kind:      uint16(evt.Kind),
		CreatedAt: evt.CreatedAt,
	}
	var e error
	var id, pub, sig []byte
	id, e = hex.Dec(string(evt.ID))
	log.D.Chk(e)
	copy(be.ID[:], id)
	pub, e = hex.Dec(evt.PubKey)
	log.D.Chk(e)
	copy(be.PubKey[:], pub)
	sig, e = hex.Dec(evt.Sig)
	copy(be.Sig[:], sig)
	log.D.Chk(e)
	return be
}

func (be *Event) ToNormalEvent() *event.T {
	id, e := eventid.NewEventID(hex.Enc(be.ID[:]))
	log.D.Chk(e)
	return &event.T{
		Tags:      be.Tags,
		Content:   be.Content,
		Kind:      kind.T(be.Kind),
		CreatedAt: be.CreatedAt,
		ID:        id,
		PubKey:    hex.Enc(be.PubKey[:]),
		Sig:       hex.Enc(be.Sig[:]),
	}
}
