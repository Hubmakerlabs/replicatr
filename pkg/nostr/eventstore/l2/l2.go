// Package l2 is a testing data store that implements a level 2 cache for events
// with a badger eventstore.
//
// This is a testing environment for building cache strategies.
package l2

import (
	"errors"
	"os"
	"sync"
	"time"

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

func (b *Backend) SaveLoop(c context.T, saveChan event.C) {
	for {
		select {
		case <-b.Ctx.Done():
			return
		case <-c.Done():
			return
		case ev := <-saveChan:
			if ev == nil {
				return
			}
			log.T.F("reviving event %s in L1", ev.ID)
			chk.T(b.L1.SaveEvent(c, ev))
		}
	}
}

func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C, err error) {
	ch = make(event.C, 6)
	var startGroup sync.WaitGroup
	// Both calls are async so fire them off at the same time
	startGroup.Add(1)
	ch1, err1 := b.L1.QueryEvents(c, f)
	startGroup.Add(1)
	ch2, err2 := b.L2.QueryEvents(c, f)
	errs := []error{err1, err2}
	// Start up a goroutine to catch evMap that need to be synced back after being
	// pruned and then match a search and pulled from the L2.
	//
	// It is necessary to use a second goroutine for this because handling the
	// returns to the caller are more important. Thus the save operation is done
	// after the query context is canceled.
	saveChan := make(event.C, 12)
	evMap := make(map[eventid.T]struct{})
	go b.SaveLoop(c, saveChan)
	// Events should not be duplicated in the return to the client, so a
	// mutex and eventid.T map will indicate if an event has already been returned.
	var evMx sync.Mutex
	var wg sync.WaitGroup
	go func() {
		go func() {
			wg.Add(1)
			defer wg.Done()
			startGroup.Done()
		out:
			for {
				select {
				case <-c.Done():
					// if context is closed, break out
					log.I.Ln("query context done")
					break out
				case <-b.Ctx.Done():
					log.I.Ln("backend context done")
					break out
				case ev1 := <-ch1:
					if ev1 == nil {
						// this means the channel has closed
						return
					}
					evMx.Lock()
					// no point in storing it if it is already found.
					// log.I.Ln(evMap)
					_, ok := evMap[ev1.ID]
					if ok {
						evMx.Unlock()
						log.I.Ln("layer 2 already returned")
						continue
					}
					evMap[ev1.ID] = struct{}{}
					evMx.Unlock()
					// if the event is missing a signature, we can ascertain that the L1 has found a
					// reference to an event but does not have possession of the event.
					if ev1.Sig != "" || ev1.ID == "" {
						// first to find should send
						log.I.Ln("sending event from l1")
						ch <- ev1
					} else {
						// spawn a query to fetch the event ID from the L2
						go b.Revive(ev1, c, ch, saveChan, &wg)
					}
				}
			}
			log.I.Ln("l1 query complete")
		}()
		go func() {
			wg.Add(1)
			defer wg.Done()
			startGroup.Done()
		out:
			for {
				select {
				case <-c.Done():
					// if context is closed, break out
					log.I.Ln("query context done")
					break out
				case <-b.Ctx.Done():
					log.I.Ln("backend context done")
					break out
				case ev2 := <-ch2:
					if ev2 == nil {
						// this means the channel has closed
						return
					}
					evMx.Lock()
					// no point in storing it if it is already found.
					// log.I.Ln(evMap)
					_, ok := evMap[ev2.ID]
					if ok {
						evMx.Unlock()
						log.I.Ln("layer 1 already returned")
						continue
					}
					evMap[ev2.ID] = struct{}{}
					evMx.Unlock()
					// first to find should send
					log.I.Ln("sending event from l2")
					ch <- ev2
				}
			}
			log.I.Ln("l2 query complete")
		}()
	}()
	startGroup.Wait()
	log.I.Ln("both layers have started query")
	// wait for both layers to terminate
	wg.Wait()
	timeout := time.After(time.Second / 100)
drain:
	for {
		select {
		case ev := <-ch:
			if ev == nil {
				break drain
			}
			log.I.Ln("event got stuck", ev.ToObject().String())
		case <-timeout:
			log.I.Ln("timeout draining")
			break drain
		}
	}
	log.I.Ln("canceling query context")
	c.Done()
	log.I.Ln("closing event results channel")
out:
	for {
		select {
		case ev := <-ch:
			log.I.Ln("drained event", ev.ToObject().String())
		default:
			break out
		}
	}
	// close(ch)
	// close(saveChan)
	err = errors.Join(errs...)
	return
}

func (b *Backend) Revive(ev1 *event.T, c context.T, ch, saveChan event.C, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
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
