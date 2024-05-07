package cache

import (
	"bytes"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Event struct {
	JSON         []byte
	lastAccessed timestamp.T
}

type events map[eventid.T]*Event

type Encoder struct {
	mx      sync.Mutex
	events  events
	pool    *sync.Pool
	average int
}

type access struct {
	eventid.T
	lastAccessed timestamp.T
	size         int
}

// accesses is a sort.Interface that sorts in descending order of lastAccessed
type accesses []access

func (s accesses) Len() int           { return len(s) }
func (s accesses) Less(i, j int) bool { return s[i].lastAccessed > s[j].lastAccessed }
func (s accesses) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// NewEncoder creates a cache.Encoder that maintains a cache of already decoded
// JSON bytes of events to prevent doubling up of both decoding work and JSON
// bytes usage for multiple concurrent requests of overlapping events by
// clients.
func NewEncoder(c context.T, maxCacheSize int,
	gcTimer time.Duration) *Encoder {

	d := &Encoder{
		events:  make(events),
		average: 8192,
	}
	d.pool = &sync.Pool{New: func() any {
		// allocate a little more than the average to minimise reallocations.
		return make([]byte, 0, d.average*120/100)
	}}
	go func() {
		log.D.Ln("starting encoder cache garbage collector")
		tick := time.NewTicker(gcTimer)
	gcLoop:
		for {
			select {
			case <-c.Done():
				log.I.Ln("terminating decoder cache garbage collector")
				return
			case <-tick.C:
				var total int
				for i := range d.events {
					d.mx.Lock()
					total += len(d.events[i].JSON)
					d.mx.Unlock()
				}
				log.W.Ln("total encode cache utilization:", total, "of", maxCacheSize,
					"average buffer size:", d.average, "count of events:", len(d.events))
				if total > maxCacheSize {
					// create list of cache by access time
					var accessed accesses
					for id := range d.events {
						d.mx.Lock()
						accessed = append(accessed,
							access{
								T:            id,
								lastAccessed: d.events[id].lastAccessed,
								size:         len(d.events[id].JSON),
							})
						d.mx.Unlock()
					}
					sort.Sort(accessed)
					var last, size int
					// count off the items in descending timestamp order until the size exceeds the
					// average.
					for ; size < maxCacheSize; last++ {
						if last >= len(accessed) {
							// no need to GC
							continue gcLoop
						}
						size += accessed[last].size
					}
					if size <= maxCacheSize || len(accessed)-last == 0 {
						continue gcLoop
					}
					log.I.F("pruning out %d events making up %d bytes of %d of cached decoded events, will be %d bytes after",
						len(accessed)-last, total-size, total, size)
					for ; last < len(accessed); last++ {
						// free the buffers so they go back to the pool
						d.pool.Put(d.events[accessed[last].T].JSON)
						// delete the map entry of the expired event json
						delete(d.events, accessed[last].T)
					}
				}
			}
		}

	}()
	return d
}

// Put stores an event's encoded JSON form for access by concurrent client
// requests. Call this with an event decoded from the database so that
// concurrent queries that match it can avoid repeated decode/allocate steps.
//
// If the json is available as well, skip re-encoding it. If the event is
// already in the cache, return the json.
func (d *Encoder) Put(ev *event.T, js []byte) (j []byte, err error) {
	d.mx.Lock()
	defer d.mx.Unlock()
	rec, have := d.events[ev.ID]
	if have {
		// if we have it, just bump the record and return the stored JSON
		rec.lastAccessed = timestamp.Now()
		j = rec.JSON
		return
	}
	if js != nil {
		// if it is already available encoded, don't re-encode it.
		j = js
	} else {
		// write the JSON to a new buffer
		buf := bytes.NewBuffer(d.pool.Get().([]byte))
		ev.ToObject().Buffer(buf)
		j = buf.Bytes()
	}
	// we don't have it so store it now
	d.events[ev.ID] = &Event{JSON: j, lastAccessed: timestamp.Now()}
	// simple moving average should avoid reallocations 50% of the time
	d.average = (d.average + len(j)) / 2
	return
}

// Get retrieves the encoded JSON for a given event if it is cached.
//
// If ok is false, then the caller needs to fetch the event from elsewhere.
func (d *Encoder) Get(evId eventid.T) (b []byte, ok bool) {
	d.mx.Lock()
	defer d.mx.Unlock()
	var e *Event
	if e, ok = d.events[evId]; !ok {
		return
	}
	b = e.JSON
	d.events[evId].lastAccessed = timestamp.Now()
	// if result is found, return ok
	ok = true
	return
}
