package nostr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/fiatjaf/generic-ristretto/z"

	"github.com/puzpuzpuz/xsync/v2"
)

const MAX_LOCKS = 50

var namedMutexPool = make([]sync.Mutex, MAX_LOCKS)

func namedLock(name string) (unlock func()) {
	idx := z.MemHashString(name) % MAX_LOCKS
	namedMutexPool[idx].Lock()
	return namedMutexPool[idx].Unlock
}

type SimplePool struct {
	Relays  map[string]*Relay
	Context context.Context

	cancel context.CancelFunc
}

type IncomingEvent struct {
	*nip1.Event
	Relay *Relay
}

func NewSimplePool(ctx context.Context) *SimplePool {
	ctx, cancel := context.WithCancel(ctx)

	return &SimplePool{
		Relays: make(map[string]*Relay),

		Context: ctx,
		cancel:  cancel,
	}
}

func (pool *SimplePool) EnsureRelay(url string) (*Relay, error) {
	nm := normalize.URL(url)

	defer namedLock(url)()

	relay, ok := pool.Relays[nm]
	if ok && relay.IsConnected() {
		// already connected, unlock and return
		return relay, nil
	} else {
		var err error
		// we use this ctx here so when the pool dies everything dies
		ctx, cancel := context.WithTimeout(pool.Context, time.Second*15)
		defer cancel()
		if relay, err = RelayConnect(ctx, nm); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}

		pool.Relays[nm] = relay
		return relay, nil
	}
}

// SubMany opens a subscription with the given filters to multiple relays
// the subscriptions only end when the context is canceled
func (pool *SimplePool) SubMany(ctx context.Context, urls []string, filters nip1.Filters, unique bool) chan IncomingEvent {
	return pool.subMany(ctx, urls, filters, true)
}

// SubManyNonUnique is like SubMany, but returns duplicate events if they come from different relays
func (pool *SimplePool) SubManyNonUnique(ctx context.Context, urls []string, filters nip1.Filters, unique bool) chan IncomingEvent {
	return pool.subMany(ctx, urls, filters, false)
}

func (pool *SimplePool) subMany(ctx context.Context, urls []string, filters nip1.Filters, unique bool) chan IncomingEvent {
	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[bool]()

	pending := xsync.NewCounter()
	initial := len(urls)
	pending.Add(int64(initial))
	for _, url := range urls {
		go func(nm string) {
			relay, err := pool.EnsureRelay(nm)
			if err != nil {
				return
			}

			sub, _ := relay.Subscribe(ctx, filters)
			if sub == nil {
				return
			}

			for evt := range sub.Events {
				stop := false
				if unique {
					_, stop = seenAlready.LoadOrStore(evt.ID.String(), true)
				}
				if !stop {
					select {
					case events <- IncomingEvent{Event: evt, Relay: relay}:
					case <-ctx.Done():
						return
					}
				}
			}

			pending.Dec()
			if pending.Value() == 0 {
				close(events)
			}
		}(normalize.URL(url))
	}

	return events
}

// SubManyEose is like SubMany, but it stops subscriptions and closes the channel when gets a EOSE
func (pool *SimplePool) SubManyEose(ctx context.Context, urls []string, filters nip1.Filters) chan IncomingEvent {
	return pool.subManyEose(ctx, urls, filters, true)
}

// SubManyEoseNonUnique is like SubManyEose, but returns duplicate events if they come from different relays
func (pool *SimplePool) SubManyEoseNonUnique(ctx context.Context, urls []string, filters nip1.Filters) chan IncomingEvent {
	return pool.subManyEose(ctx, urls, filters, false)
}

func (pool *SimplePool) subManyEose(ctx context.Context, urls []string, filters nip1.Filters, unique bool) chan IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)

	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[bool]()
	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	go func() {
		// this will happen when all subscriptions get an eose (or when they die)
		wg.Wait()
		cancel()
		close(events)
	}()

	for _, url := range urls {
		go func(nm string) {
			defer wg.Done()

			relay, err := pool.EnsureRelay(nm)
			if err != nil {
				return
			}

			sub, err := relay.Subscribe(ctx, filters)
			if sub == nil {
				log.E.F("error subscribing to %s with %v: %s", relay, filters, err)
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-sub.EndOfStoredEvents:
					return
				case evt, more := <-sub.Events:
					if !more {
						return
					}

					stop := false
					if unique {
						_, stop = seenAlready.LoadOrStore(evt.ID.String(), true)
					}
					if !stop {
						select {
						case events <- IncomingEvent{Event: evt, Relay: relay}:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}(normalize.URL(url))
	}

	return events
}

// QuerySingle returns the first event returned by the first relay, cancels everything else.
func (pool *SimplePool) QuerySingle(ctx context.Context, urls []string, filter *nip1.Filter) *IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for ievt := range pool.SubManyEose(ctx, urls, nip1.Filters{filter}) {
		return &ievt
	}
	return nil
}
