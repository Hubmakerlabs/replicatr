package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
)

func (b *Backend) Wipe() (err error) {
	if err = b.DB.DropPrefix([][]byte{
		{index.Event.B()},
		{index.CreatedAt.B()},
		{index.Id.B()},
		{index.Kind.B()},
		{index.Pubkey.B()},
		{index.PubkeyKind.B()},
		{index.Tag.B()},
		{index.Tag32.B()},
		{index.TagAddr.B()},
		{index.Counter.B()},
	}...); chk.E(err) {
		return
	}
	// if err = b.DB.RunValueLogGC(0.8); chk.E(err) {
	// 	return
	// }
	return
}
