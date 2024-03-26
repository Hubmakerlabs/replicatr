package app

import (
	"encoding/base64"
	"os"
	"sync"
	"time"

	"lukechampine.com/frand"
	"mleku.dev/git/interrupt"
	"mleku.dev/git/nostr/client"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/eventid"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/filters"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/nostrbinary"
	"mleku.dev/git/nostr/tag"
	"mleku.dev/git/nostr/timestamp"
	"mleku.dev/git/qu"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Config struct {
	MaxContentSize int    `arg:"-m,--maxsize" default:"1024" help:"maximum size of event content field to generate"`
	TotalSize      int    `arg:"-t,--totalsize" help:"total amount of events data (binary) in bytes to generate"`
	MaxDelay       int    `arg:"-d,--maxdelay" default:"30" help:"max delay between dispatching events (milliseconds)"`
	Nsec           string `arg:"-n,--nsec" help:"secret key in hex or bech32 nsec format to use for signing events and auth"`
	Relay          string `arg:"-r,--relay" default:"ws://127.0.0.1:3334" help:"relay to dispatch to, eg ws://127.0.0.1:3334"`
}

type Counter struct {
	id        *eventid.T
	requested int
}

var (
	Sec     string
	mx      sync.Mutex
	counter []Counter
	total   int
)

func (cfg *Config) Main() (err error) {
	log.I.Ln("running firehose")
	c, cancel := context.Cancel(context.Bg())
	interrupt.AddHandler(func() {
		cancel()
		os.Exit(0)
	})
	// connect to a relay
	var rl *client.T
	if rl, err = client.ConnectWithAuth(c, cfg.Relay, Sec); chk.E(err) {
		return
	}
	newEvent := qu.T()
	go func() {
		ticker := time.NewTicker(time.Second)
		var fetchIDs []*eventid.T
		// start fetching loop
		for {
			select {
			case <-newEvent:
				// make new request, not necessarily from existing... bias rng
				// factor by request count
				mx.Lock()
				for i := range counter {
					rn := frand.Intn(256)
					// multiply this number by the number of accesses the event
					// has and request every event that gets over 50% so that we
					// create a bias towards already requested.
					if counter[i].requested+rn > 192 {
						log.I.Ln("counter", counter[i].requested, "+", rn, "=",
							counter[i].requested+rn)
						// log.T.Ln("adding to fetchIDs")
						counter[i].requested++
						fetchIDs = append(fetchIDs, counter[i].id)
					}
				}
				log.W.Ln("fetchIDs", len(fetchIDs), fetchIDs)
				mx.Unlock()
			case <-ticker.C:
				// copy out current list of events to request
				mx.Lock()
				log.W.Ln("ticker", len(fetchIDs))
				ids := make(tag.T, len(fetchIDs))
				for i := range fetchIDs {
					ids[i] = fetchIDs[i].String()
				}
				fetchIDs = fetchIDs[:0]
				mx.Unlock()
				if len(ids) > 0 {
					for i := range ids {
						go func(i int) {
							sc, _ := context.Timeout(c, 2*time.Second)
							sub := rl.PrepareSubscription(sc, filters.T{&filter.T{
								IDs: tag.T{ids[i]},
							}})
							if err = sub.Fire(); chk.E(err) {
								return
							}
							go func() {
								// receive the results
								select {
								case <-sub.Events:
									log.I.Ln("received event")
								case <-sub.EndOfStoredEvents:
									log.I.Ln("EOSE")
								case <-sc.Done():
									log.I.Ln("subscription done")
								}
							}()
						}(i)
					}
				}
			case <-c.Done():
				return
			}
		}
	}()
	var ev *event.T
	var bs int
out:
	for {
		select {
		case <-c.Done():
			return
		default:
		}
		if total > cfg.TotalSize {
			break out
		}
		if ev, bs, err = GenerateEvent(Sec, cfg.MaxContentSize); chk.E(err) {
			return
		}
		mx.Lock()
		counter = append(counter, Counter{id: &ev.ID, requested: 1})
		total += bs
		if total > cfg.TotalSize {
			mx.Unlock()
			break out
		}
		mx.Unlock()
		newEvent.Signal()
		sc, _ := context.Timeout(c, 2*time.Second)
		if err = rl.Publish(sc, ev); chk.E(err) {
		}
		delay := frand.Intn(cfg.MaxDelay)
		log.I.Ln("waiting between", delay, "ms")
		if delay == 0 {
			continue
		}
		select {
		case <-c.Done():
			return
		case <-time.After(time.Duration(delay) * time.Millisecond):
		}
	}
	select {
	case <-c.Done():
	}
	return
}

func GenerateEvent(nsec string, maxSize int) (ev *event.T, binSize int, err error) {
	l := frand.Intn(maxSize * 6 / 8) // account for base64 expansion
	ev = &event.T{
		Kind:      kind.TextNote,
		CreatedAt: timestamp.Now(),
		Content:   base64.StdEncoding.EncodeToString(frand.Bytes(l)),
	}
	if err = ev.Sign(nsec); chk.E(err) {
		return
	}
	var bin []byte
	bin, err = nostrbinary.Marshal(ev)
	binSize = len(bin)
	return
}
