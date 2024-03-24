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
	"mleku.dev/git/nostr/filters"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/nostrbinary"
	"mleku.dev/git/nostr/subscription"
	"mleku.dev/git/nostr/tag"
	"mleku.dev/git/nostr/timestamp"
	"mleku.dev/git/qu"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Config struct {
	MaxContentSize int    `arg:"-m,--maxsize" default:"1024" help:"maximum size of event content field to generate"`
	TotalSize      int    `arg:"-t,--totalsize" help:"total amount of events data (binary) in bytes to generate"`
	MaxDelay       int    `arg:"-d,--maxdelay" default:"3" help:"max delay between dispatching events (seconds)"`
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
	})
	// connect to a relay
	var rl *client.T
	if rl, err = client.ConnectWithAuth(c, cfg.Relay, Sec); chk.E(err) {
		return
	}
	newEvent := qu.T()
	go func() {
		ticker := time.NewTicker(time.Second)
		// start fetching loop
		for {
			var fetchIDs []*eventid.T
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
					if counter[i].requested*rn > 192 {
						fetchIDs = append(fetchIDs, counter[i].id)
					}
				}
				mx.Unlock()
			case <-ticker.C:
				// copy out current list of events to request (this is a single
				// thread we don't need to mutex)
				f := make([]*eventid.T, len(fetchIDs))
				copy(f, fetchIDs)
				fetchIDs = fetchIDs[:0]
				// because we copied it and purged the original so it can
				// refill, we can now run the actual fetch in the background as
				// it is only to poke the database.
				go func(f []*eventid.T) {
					limit := len(f)
					ids := make(tag.T, limit)
					for i := range f {
						ids[i] = f[i].String()
					}
					filt := filters.T{
						{IDs: ids, Limit: &limit},
					}
					var sub *subscription.T
					if sub, err = rl.Subscribe(c, filt); chk.E(err) {
						// not sure what to do here
					}
					// receive and discard, we are only doing this to make the
					// relay increment the access counters.
				out:
					for {
						select {
						case <-c.Done():
							break out
						case <-sub.EndOfStoredEvents:
							sub.Unsub()
							break out
						case _, more := <-sub.Events:
							if !more {
								break out
							}
						}
					}
				}(f)
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
			break out
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
		if err = rl.Publish(c, ev); chk.E(err) {
		}
		log.I.Ln("waiting between")
		delay := frand.Intn(cfg.MaxDelay)
		if delay == 0 {
			continue
		}
		select {
		case <-c.Done():
			break out
		case <-time.After(time.Duration(delay) * time.Second):
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
