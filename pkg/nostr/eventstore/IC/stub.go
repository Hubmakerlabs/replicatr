package IC

import (
	"os"

	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/eventstore/badger"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/kinds"
	"mleku.dev/git/nostr/relayinfo"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Backend struct {
	// Badger backend must populated
	Badger *badger.Backend
}

// for now this is just a stub that calls all of the badger.Backend methods,
// later this will include the ICP storage driver functionality.

func (b *Backend) Init(inf *relayinfo.T) (err error) {
	if err = b.Badger.Init(inf); chk.D(err) {
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
func (b *Backend) CountEvents(c context.T, f *filter.T) (count int64, err error) {
	var forBadger, forIC kinds.T
	for i := range f.Kinds {
		if kinds.IsPrivileged(f.Kinds[i]) {
			forBadger = append(forBadger, f.Kinds[i])
		} else {
			forIC = append(forIC, f.Kinds[i])
		}
	}
	icFilter := f.Duplicate()
	f.Kinds = forBadger
	icFilter.Kinds = forIC
	if count, err = b.Badger.CountEvents(c, f); chk.E(err) {
		return
	}
	// todo: this will be changed to call the IC count events implementation
	return b.Badger.CountEvents(c, icFilter)
}
func (b *Backend) DeleteEvent(c context.T, evt *event.T) (err error) {
	if kinds.IsPrivileged(evt.Kind) {
		return b.Badger.DeleteEvent(c, evt)
	}
	// todo: this will be the IC store
	return b.Badger.DeleteEvent(c, evt)
}
func (b *Backend) QueryEvents(c context.T, C chan *event.T, f *filter.T) (err error) {
	var forBadger, forIC kinds.T
	for i := range f.Kinds {
		if kinds.IsPrivileged(f.Kinds[i]) {
			forBadger = append(forBadger, f.Kinds[i])
		} else {
			forIC = append(forIC, f.Kinds[i])
		}
	}
	icFilter := f.Duplicate()
	f.Kinds = forBadger
	icFilter.Kinds = forIC
	ch := make(chan *event.T)
	if err = b.Badger.QueryEvents(c, ch, icFilter); chk.E(err) {
		return
	}
	// todo: this will be changed to the IC query events function
	return b.Badger.QueryEvents(c, ch, f)
}
func (b *Backend) SaveEvent(c context.T, evt *event.T) (err error) {
	if kinds.IsPrivileged(evt.Kind) {
		return b.Badger.DeleteEvent(c, evt)
	}
	// todo: this will be the IC store
	return b.Badger.SaveEvent(c, evt)
}
