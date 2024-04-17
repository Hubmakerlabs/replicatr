package count

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/sizer"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

var Prefix = []byte{byte(index.Counter)}

type Item struct {
	Serial    []byte
	Size      uint32
	Freshness timestamp.T
}

type Items []*Item

func MakeItem(ser *serial.T, ts *createdat.T,
	size *sizer.T) *Item {

	return &Item{
		Serial:    ser.Val,
		Freshness: ts.Val,
		Size:      size.Val,
	}
}

func (c Items) Len() int           { return len(c) }
func (c Items) Less(i, j int) bool { return c[i].Freshness < c[j].Freshness }
func (c Items) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c Items) Total() (total int) {
	for i := range c {
		total += int(c[i].Size)
	}
	return
}
