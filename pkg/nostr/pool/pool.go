package pool

import (
	"fmt"
	"hash/maphash"
	"sync"
	"time"
	"unsafe"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
	"github.com/fiatjaf/generic-ristretto/z"
	"github.com/puzpuzpuz/xsync/v2"
)

var log = slog.GetStd()

const MAX_LOCKS = 50

type Option interface {
	IsPoolOption()
	Apply(*Simple)
}

// WithAuthHandler must be a function that signs the auth event when called.
// it will be called whenever any relay in the pool returns a `CLOSED` message
// with the "auth-required:" prefix, only once for each relay
type WithAuthHandler func(authEvent *event.T) error

func (_ WithAuthHandler) IsPoolOption() {}
func (h WithAuthHandler) Apply(pool *Simple) {
	pool.authHandler = h
}

var _ Option = (WithAuthHandler)(nil)

func PointerHasher(_ maphash.Seed, k string) uint64 {
	return uint64(uintptr(unsafe.Pointer(&k)))
}

var namedMutexPool = make([]sync.Mutex, MAX_LOCKS)

func namedLock(name string) (unlock func()) {
	idx := z.MemHashString(name) % MAX_LOCKS
	namedMutexPool[idx].Lock()
	return namedMutexPool[idx].Unlock
}

type Simple struct {
	Relays      *xsync.MapOf[string, *relay.T]
	authHandler func(*event.T) error
	Context     context.T
	cancel      context.F
}

type IncomingEvent struct {
	Event *event.T
	Relay *relay.T
}

func NewSimplePool(c context.T, opts ...Option) (p *Simple) {
	c, cancel := context.Cancel(c)

	p = &Simple{
		Relays:  xsync.NewTypedMapOf[string, *relay.T](PointerHasher),
		Context: c,
		cancel:  cancel,
	}

	for _, opt := range opts {
		opt.Apply(p)
	}

	return
}

func (p *Simple) EnsureRelay(url string) (rl *relay.T, err error) {
	nm := normalize.URL(url)

	defer namedLock(url)()
	var ok bool
	rl, ok = p.Relays.Load(nm)
	if ok && rl.IsConnected() {
		// already connected, unlock and return
		return rl, nil
	} else {
		// we use this ctx here so when the pool dies everything dies
		c, cancel := context.Timeout(p.Context, time.Second*15)
		defer cancel()
		if rl, err = relay.Connect(c, nm); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		p.Relays.Store(nm, rl)
		return
	}
}

// SubMany opens a subscription with the given filters to multiple relays
// the subscriptions only end when the context is canceled
func (p *Simple) SubMany(c context.T, urls []string, f filters.T,
	unique bool) chan IncomingEvent {

	return p.subMany(c, urls, f, unique)
}

// SubManyNonUnique is like SubMany, but returns duplicate events if they come from different relays
func (p *Simple) SubManyNonUnique(c context.T, urls []string, filters filters.T, unique bool) chan IncomingEvent {
	return p.subMany(c, urls, filters, false)
}

func (p *Simple) subMany(c context.T, urls []string, filters filters.T, unique bool) chan IncomingEvent {
	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[bool]()

	pending := xsync.NewCounter()
	initial := len(urls)
	pending.Add(int64(initial))
	for _, url := range urls {
		go func(nm string) {
			rl, err := p.EnsureRelay(nm)
			if err != nil {
				return
			}

			sub, _ := rl.Subscribe(c, filters)
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
					case events <- IncomingEvent{Event: evt, Relay: rl}:
					case <-c.Done():
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

// SubManyEose is like SubMany, but it stops subscriptions and closes the
// channel when gets a EOSE
func (p *Simple) SubManyEose(c context.T, urls []string, f filters.T, unique bool) chan IncomingEvent {
	return p.subManyEose(c, urls, f, true)
}

// SubManyEoseNonUnique is like SubManyEose, but returns duplicate events if
// they come from different relays
func (p *Simple) SubManyEoseNonUnique(c context.T, urls []string, f filters.T, unique bool) chan IncomingEvent {
	return p.subManyEose(c, urls, f, false)
}

func (p *Simple) subManyEose(c context.T, urls []string, f filters.T, unique bool) chan IncomingEvent {
	c, cancel := context.Cancel(c)

	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[bool]()
	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	go func() {
		// this will happen when all subscriptions get an eose (or when they
		// die)
		wg.Wait()
		cancel()
		close(events)
	}()

	for _, url := range urls {
		go func(nm string) {
			defer wg.Done()

			rl, err := p.EnsureRelay(nm)
			if err != nil {
				return
			}

			sub, err := rl.Subscribe(c, f)
			if sub == nil {
				log.E.F("error subscribing to %s with %v: %s", rl, f, err)
				return
			}

			for {
				select {
				case <-c.Done():
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
						case events <- IncomingEvent{Event: evt, Relay: rl}:
						case <-c.Done():
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
func (p *Simple) QuerySingle(c context.T, urls []string, f *filter.T, unique bool) *IncomingEvent {
	c, cancel := context.Cancel(c)
	defer cancel()
	for ievt := range p.SubManyEose(c, urls, filters.T{f}, true) {
		return &ievt
	}
	return nil
}
