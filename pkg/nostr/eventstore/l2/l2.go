// Package l2 is a testing data store that implements a level 2 cache for events
// with a badger eventstore.
//
// This is a testing environment for building cache strategies.
package l2

import (
	"errors"
	"os"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Backend struct {
	Ctx context.T
	WG  *sync.WaitGroup
	// L1 is a primary, presumably local store, which should be faster, but may be
	// space constrained.
	L1 *badger.Backend
	// L2 is a secondary, possibly slower but bigger cache. It could be an IC
	// canister, an IPFS based store with an indexing spider or indeed a giant
	// spinning disk.
	L2 eventstore.Store
}

func (b *Backend) Init() (err error) {
	if err = b.L1.Init(); chk.E(err) {
		return
	}
	if err = b.L2.Init(); chk.E(err) {
		return
	}
	return
}

func (b *Backend) Close() {
	b.L1.Close()
	b.L2.Close()
	return
}

func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C, err error) {
	ch = make(event.C)
	// Both calls are async so fire them off at the same time
	ch1, err1 := b.L1.QueryEvents(c, f)
	ch2, err2 := b.L2.QueryEvents(c, f)
	// Start up a goroutine to catch evMap that need to be synced back after being
	// pruned and then match a search and pulled from the L2.
	//
	// It is necessary to use a second goroutine for this because handling the
	// returns to the caller are more important. Thus the save operation is done
	// after the query context is canceled.
	saveChan := make(event.C)
	evMap := make(map[*eventid.T]struct{})
	go func() {
		// var saveEvents []*event.T
		for {
			select {
			case <-c.Done():
				// for _, ev := range saveEvents {
				// 	log.I.F("reviving event %s in L1", ev.ID)
				// 	chk.E(b.L1.SaveEvent(c, ev))
				// }
				return
			case ev := <-saveChan:
				if ev == nil {
					continue
				}
				log.T.F("reviving event %s in L1", ev.ID)
				chk.T(b.L1.SaveEvent(c, ev))
				// saveEvents = append(saveEvents, ev)
			}
		}
	}()
	// Events should not be duplicated in the return to the client, so a
	// mutex and eventid.T map will indicate if an event has already been returned.
	var evMx sync.Mutex
	errs := []error{err1, err2}
	go func() {
		defer close(ch)
	out:
		for {
			select {
			case <-c.Done():
				// if context is closed, break out
				break out
			case ev1 := <-ch1:
				if ev1 == nil {
					// log.W.Ln("nil event")
					break
				}
				evMx.Lock()
				// no point in storing it if it is already found.
				// log.I.Ln(evMap)
				_, ok := evMap[&ev1.ID]
				if ok {
					evMx.Unlock()
					continue
				}
				evMap[&ev1.ID] = struct{}{}
				evMx.Unlock()
				// if the event is missing a signature, we can ascertain that the L1 has found a
				// reference to an event but does not have possession of the event.
				if ev1.Sig == "" && ev1.ID != "" {
					// spawn a query to fetch the event ID from the L2
					go func(ev1 *event.T) {
						log.T.F("retrieving event %s from L2", ev1.ID)
						ch3, err3 := b.L2.QueryEvents(c,
							&filter.T{IDs: tag.T{ev1.ID.String()}})
						chk.E(err3)
					out2:
						for {
							select {
							case <-c.Done():
								// if context is closed, break out
								break out2
							case ev3 := <-ch3:
								ch <- ev3
								// need to queue up the event to restore the event and counter records
								saveChan <- ev3
								// there can only be one
								break out2
							}
						}
					}(ev1)
				} else {
					// first to find should send
					ch <- ev1
				}
			case ev2 := <-ch2:
				if ev2 == nil {
					// log.W.Ln("nil event")
					break
				}
				evMx.Lock()
				// no point in storing it if it is already found.
				// log.I.Ln(evMap)
				_, ok := evMap[&ev2.ID]
				if ok {
					evMx.Unlock()
					continue
				}
				evMap[&ev2.ID] = struct{}{}
				evMx.Unlock()
				// first to find should send
				ch <- ev2
			}
		}
	}()
	err = errors.Join(errs...)
	return
}

func (b *Backend) CountEvents(c context.T, f *filter.T) (count int, err error) {
	count1, err1 := b.L1.CountEvents(c, f)
	count2, err2 := b.L2.CountEvents(c, f)
	// we return the maximum, it is assumed the L2 is authoritative, but it could be
	// the L1 has more for whatever reason, so return the maximum of the two.
	count = count1
	if count2 > count {
		count = count2
	}
	err = errors.Join(err1, err2)
	return
}

// DeleteEvent deletes the event if found. If not found, will return
// eventstore.ErrEventNotExists.
//
// Relay may have filters to block this, by default only an event author can
// delete an event, but this is not processed here, it must be done in a
// previous step.
func (b *Backend) DeleteEvent(c context.T, ev *event.T) (err error) {
	// delete the events
	err = errors.Join(b.L1.DeleteEvent(c, ev), b.L2.DeleteEvent(c, ev))
	return
}

// SaveEvent stores the event to the local badger store and the L2, and returns
// any errors from each store. The only error defined here is
// eventstore.ErrDupEvent if the store already has the event.
//
// Any errors from this method are not fatal, mostly, mostly anything else, like
// auth- or filter- related denials are from a separate subsystem.
func (b *Backend) SaveEvent(c context.T, ev *event.T) (err error) {
	err = errors.Join(b.L1.SaveEvent(c, ev), b.L2.SaveEvent(c, ev))
	return
}
