package count

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

type Item struct {
	Serial    uint64
	Size      uint32
	Freshness timestamp.T
}

type Items []*Item

func (c Items) Len() int           { return len(c) }
func (c Items) Less(i, j int) bool { return c[i].Freshness < c[j].Freshness }
func (c Items) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c Items) Total() (total int) {
	for i := range c {
		total += int(c[i].Size)
	}
	return
}
