package badger

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
)

func (b *Backend) Wipe() (err error) {
	if err = b.DB.DropPrefix([][]byte{
		{index.Event.Byte()},
		{index.CreatedAt.Byte()},
		{index.Id.Byte()},
		{index.Kind.Byte()},
		{index.Pubkey.Byte()},
		{index.PubkeyKind.Byte()},
		{index.Tag.Byte()},
		{index.Tag32.Byte()},
		{index.TagAddr.Byte()},
		{index.Counter.Byte()},
	}...); chk.E(err) {
		return
	}
	// if err = b.DB.RunValueLogGC(0.8); chk.E(err) {
	// 	return
	// }
	return
}
