package pools

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/puzpuzpuz/xsync/v2"
)

const (
	seenAlreadyDropTick = time.Minute
)

type SimplePool struct {
	Relays  *xsync.MapOf[string, *relays.Relay]
	Context context.Context

	authHandler func(*event.T) error
	cancel      context.CancelFunc
}

type IncomingEvent struct {
	*event.T
	Relay *relays.Relay
}

type PoolOption interface {
	IsPoolOption()
	Apply(*SimplePool)
}

func NewSimplePool(ctx context.Context, opts ...PoolOption) *SimplePool {
	ctx, cancel := context.WithCancel(ctx)

	pool := &SimplePool{
		Relays: xsync.NewMapOf[*relays.Relay](),

		Context: ctx,
		cancel:  cancel,
	}

	for _, opt := range opts {
		opt.Apply(pool)
	}

	return pool
}

// WithAuthHandler must be a function that signs the auth event when called.
// it will be called whenever any relay in the pool returns a `CLOSED` message
// with the "auth-required:" prefix, only once for each relay
type WithAuthHandler func(authEvent *event.T) error

func (_ WithAuthHandler) IsPoolOption() {}
func (h WithAuthHandler) Apply(pool *SimplePool) {
	pool.authHandler = h
}

var _ PoolOption = (WithAuthHandler)(nil)

func (pool *SimplePool) EnsureRelay(url string) (*relays.Relay, error) {
	nm := normalize.URL(url)

	defer NamedLock(url)()

	rl, ok := pool.Relays.Load(nm)
	if ok && rl.IsConnected() {
		// already connected, unlock and return
		return rl, nil
	} else {
		var e error
		// we use this ctx here so when the pool dies everything dies
		ctx, cancel := context.WithTimeout(pool.Context, time.Second*15)
		defer cancel()
		if rl, e = relays.RelayConnect(ctx, nm); e != nil {
			return nil, fmt.Errorf("failed to connect: %w", e)
		}

		pool.Relays.Store(nm, rl)
		return rl, nil
	}
}

// SubMany opens a subscription with the given filters to multiple relays
// the subscriptions only end when the context is canceled
func (pool *SimplePool) SubMany(ctx context.Context, urls []string, filters filters.T) chan IncomingEvent {
	return pool.subMany(ctx, urls, filters, true)
}

// SubManyNonUnique is like SubMany, but returns duplicate events if they come from different relays
func (pool *SimplePool) SubManyNonUnique(ctx context.Context, urls []string, filters filters.T) chan IncomingEvent {
	return pool.subMany(ctx, urls, filters, false)
}

func (pool *SimplePool) subMany(ctx context.Context, urls []string, filters filters.T, unique bool) chan IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // do this so `go vet` will stop complaining
	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[timestamp.Timestamp]()
	ticker := time.NewTicker(seenAlreadyDropTick)

	eose := false

	pending := xsync.NewCounter()
	pending.Add(int64(len(urls)))
	for _, url := range urls {
		go func(nm string) {
			defer func() {
				pending.Dec()
				if pending.Value() == 0 {
					close(events)
				}
				cancel()
			}()

			hasAuthed := false
			interval := 3 * time.Second
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				var sub *relays.Subscription

				rl, e := pool.EnsureRelay(nm)
				if e != nil {
					goto reconnect
				}
				hasAuthed = false

			subscribe:
				sub, e = rl.Subscribe(ctx, filters)
				if e != nil {
					goto reconnect
				}

				go func() {
					<-sub.EndOfStoredEvents
					eose = true
				}()

				// reset interval when we get a good subscription
				interval = 3 * time.Second

				for {
					select {
					case evt, more := <-sub.Events:
						if !more {
							// this means the connection was closed for weird reasons, like the server shut down
							// so we will update the filters here to include only events seem from now on
							// and try to reconnect until we succeed
							now := timestamp.Now()
							for i := range filters {
								filters[i].Since = &now
							}
							goto reconnect
						}
						if unique {
							if _, seen := seenAlready.LoadOrStore(evt.ID, evt.CreatedAt); seen {
								continue
							}
						}
						select {
						case events <- IncomingEvent{T: evt, Relay: rl}:
						case <-ctx.Done():
						}
					case <-ticker.C:
						if eose {
							old := timestamp.Timestamp(time.Now().Add(-seenAlreadyDropTick).Unix())
							seenAlready.Range(func(id string, value timestamp.Timestamp) bool {
								if value < old {
									seenAlready.Delete(id)
								}
								return true
							})
						}
					case reason := <-sub.ClosedReason:
						if strings.HasPrefix(reason, "auth-required:") && pool.authHandler != nil && !hasAuthed {
							// rl is requesting auth. if we can we will perform auth and try again
							if e := rl.Auth(ctx, pool.authHandler); e == nil {
								hasAuthed = true // so we don't keep doing AUTH again and again
								goto subscribe
							}
						} else {
							log.Printf("CLOSED from %s: '%s'\n", nm, reason)
						}
						return
					case <-ctx.Done():
						return
					}
				}

			reconnect:
				// we will go back to the beginning of the loop and try to connect again and again
				// until the context is canceled
				time.Sleep(interval)
				interval = interval * 17 / 10 // the next time we try we will wait longer
			}
		}(normalize.URL(url))
	}

	return events
}

// SubManyEose is like SubMany, but it stops subscriptions and closes the channel when gets a EOSE
func (pool *SimplePool) SubManyEose(ctx context.Context, urls []string, filters filters.T) chan IncomingEvent {
	return pool.subManyEose(ctx, urls, filters, true)
}

// SubManyEoseNonUnique is like SubManyEose, but returns duplicate events if they come from different relays
func (pool *SimplePool) SubManyEoseNonUnique(ctx context.Context, urls []string, filters filters.T) chan IncomingEvent {
	return pool.subManyEose(ctx, urls, filters, false)
}

func (pool *SimplePool) subManyEose(ctx context.Context, urls []string, filters filters.T, unique bool) chan IncomingEvent {
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

			rl, e := pool.EnsureRelay(nm)
			if e != nil {
				return
			}

			hasAuthed := false

		subscribe:
			sub, e := rl.Subscribe(ctx, filters)
			if sub == nil {
				fmt.Printf("error subscribing to %s with %v: %s", rl, filters, e)
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-sub.EndOfStoredEvents:
					return
				case reason := <-sub.ClosedReason:
					if strings.HasPrefix(reason, "auth-required:") && pool.authHandler != nil && !hasAuthed {
						// relay is requesting auth. if we can we will perform auth and try again
						e := rl.Auth(ctx, pool.authHandler)
						if e == nil {
							hasAuthed = true // so we don't keep doing AUTH again and again
							goto subscribe
						}
					}
					log.Printf("CLOSED from %s: '%s'\n", nm, reason)
					return
				case evt, more := <-sub.Events:
					if !more {
						return
					}

					if unique {
						if _, seen := seenAlready.LoadOrStore(evt.ID, true); seen {
							continue
						}
					}

					select {
					case events <- IncomingEvent{T: evt, Relay: rl}:
					case <-ctx.Done():
						return
					}
				}
			}
		}(normalize.URL(url))
	}

	return events
}

// QuerySingle returns the first event returned by the first relay, cancels everything else.
func (pool *SimplePool) QuerySingle(ctx context.Context, urls []string, f filter.T) *IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for ievt := range pool.SubManyEose(ctx, urls, filters.T{f}) {
		return &ievt
	}
	return nil
}
