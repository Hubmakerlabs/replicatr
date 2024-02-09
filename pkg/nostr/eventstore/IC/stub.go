package IC

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"mleku.online/git/slog"
)

type Backend struct {
	// Badger backend must populated
	Badger *badger.Backend
}

// for now this is just a stub that calls all of the badger.Backend methods,
// later this will include the ICP storage driver functionality.

func (b *Backend) Init() (err error) {
	if b.Badger.Log == nil || b.Badger.Check == nil {
		b.Badger.Log, b.Badger.Check = slog.New(os.Stderr)
	}
	if err = b.Badger.Init(); b.Badger.Check.D(err) {
		return
	}

	return
}
func (b *Backend) Close() {
	b.Badger.Close()
}
func (b *Backend) Serial() []byte {
	return b.Badger.Serial()
}
func (b *Backend) CountEvents(c context.T, f *filter.T) (int64, error) {
	return b.Badger.CountEvents(c, f)
}
func (b *Backend) DeleteEvent(c context.T, evt *event.T) (err error) {
	return b.Badger.DeleteEvent(c, evt)
}
func (b *Backend) QueryEvents(c context.T, f *filter.T) (chan *event.T, error) {
	return b.Badger.QueryEvents(c, f)
}
func (b *Backend) SaveEvent(c context.T, evt *event.T) (err error) {
	return b.Badger.SaveEvent(c, evt)
}
