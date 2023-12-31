package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/fiatjaf/generic-ristretto/z"

	"github.com/puzpuzpuz/xsync/v2"
)

var log, fails = log2.GetStd()

const MAX_LOCKS = 50

var namedMutexPool = make([]sync.Mutex, MAX_LOCKS)

func namedLock(name string) (unlock func()) {
	idx := z.MemHashString(name) % MAX_LOCKS
	namedMutexPool[idx].Lock()
	return namedMutexPool[idx].Unlock
}

type SimplePool struct {
	Relays  map[string]*relay.Relay
	Context context.Context

	cancel context.CancelFunc
}

type IncomingEvent struct {
	*event.T
	Relay *relay.Relay
}

func NewSimplePool(ctx context.Context) *SimplePool {
	ctx, cancel := context.WithCancel(ctx)

	return &SimplePool{
		Relays: make(map[string]*relay.Relay),

		Context: ctx,
		cancel:  cancel,
	}
}

func (p *SimplePool) EnsureRelay(url string) (*relay.Relay, error) {
	nm := normalize.URL(url)

	defer namedLock(url)()

	rl, ok := p.Relays[nm]
	if ok && rl.IsConnected() {
		// already connected, unlock and return
		return rl, nil
	} else {
		var e error
		// we use this ctx here so when the pool dies everything dies
		ctx, cancel := context.WithTimeout(p.Context, time.Second*15)
		defer cancel()
		if rl, e = relay.RelayConnect(ctx, nm); e != nil {
			return nil, fmt.Errorf("failed to connect: %w", e)
		}

		p.Relays[nm] = rl
		return rl, nil
	}
}

// SubMany opens a subscription with the given filters to multiple relays
// the subscriptions only end when the context is canceled
func (p *SimplePool) SubMany(ctx context.Context, urls []string, filters filters.T, unique bool) chan IncomingEvent {
	return p.subMany(ctx, urls, filters, true)
}

// SubManyNonUnique is like SubMany, but returns duplicate events if they come from different relays
func (p *SimplePool) SubManyNonUnique(ctx context.Context, urls []string, filters filters.T, unique bool) chan IncomingEvent {
	return p.subMany(ctx, urls, filters, false)
}

func (p *SimplePool) subMany(ctx context.Context, urls []string, filters filters.T, unique bool) chan IncomingEvent {
	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[bool]()

	pending := xsync.NewCounter()
	initial := len(urls)
	pending.Add(int64(initial))
	for _, url := range urls {
		go func(nm string) {
			rl, e := p.EnsureRelay(nm)
			if e != nil {
				return
			}

			sub, _ := rl.Subscribe(ctx, filters)
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
					case events <- IncomingEvent{T: evt, Relay: rl}:
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
func (p *SimplePool) SubManyEose(ctx context.Context, urls []string, filters filters.T) chan IncomingEvent {
	return p.subManyEose(ctx, urls, filters, true)
}

// SubManyEoseNonUnique is like SubManyEose, but returns duplicate events if they come from different relays
func (p *SimplePool) SubManyEoseNonUnique(ctx context.Context, urls []string, filters filters.T) chan IncomingEvent {
	return p.subManyEose(ctx, urls, filters, false)
}

func (p *SimplePool) subManyEose(ctx context.Context, urls []string, filters filters.T, unique bool) chan IncomingEvent {
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

			rl, e := p.EnsureRelay(nm)
			if e != nil {
				return
			}

			sub, e := rl.Subscribe(ctx, filters)
			if sub == nil {
				log.E.F("error subscribing to %s with %v: %s", rl, filters, e)
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
						case events <- IncomingEvent{T: evt, Relay: rl}:
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
func (p *SimplePool) QuerySingle(ctx context.Context, urls []string, f *filter.T) *IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for ievt := range p.SubManyEose(ctx, urls, filters.T{f}) {
		return &ievt
	}
	return nil
}
