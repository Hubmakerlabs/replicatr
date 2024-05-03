package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/ic/agent"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

func createRandomEvent(i int) (e *event.T) {
	e = &event.T{
		CreatedAt: timestamp.T(time.Now().Unix()),
		Kind:      kind.T(rand.Intn(500)),
		Tags:      []tag.T{{"tag1", "tag2"}, {"tag3"}},
		Content:   fmt.Sprintf("This is a random event content %d", i),
	}

	err := e.Sign(keys.GeneratePrivateKey())
	if err != nil {
		log.E.F("unable to create random event number %d: %v", i, err)
	}

	return
}

func main() { // arg1 = portnum, arg2 = canisterID
	if len(os.Args) < 3 {
		fmt.Println("not enough args: 2 args required <canisterURL> <canisterID>")
	}
	// Initialize the agent with the configuration for a local replica
	a, err := agent.New(context.Bg(), os.Args[2], os.Args[1], os.Args[3])
	if err != nil {
		log.E.F("failed to initialize agent: %v\n", err)
		return
	}
	// Create and save random events
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			log.D.Ln("creating random event")
			ev := createRandomEvent(i)
			if err = a.SaveEvent(ev); chk.E(err) {
				log.E.F("Failed to save event %d: %v", i, err)
				wg.Done()
				return
			}
			log.I.F("Event %s saved successfully", ev.ID)
			wg.Done()
		}(i)
	}
	wg.Wait()
	log.I.Ln("retrieving results")
	// Create a filter to query events
	s := timestamp.Now() - 24*60*60 // for one day before now
	since := s.Ptr()
	until := timestamp.Now().Ptr()
	l := 100
	limit := &l

	f := &filter.T{
		Since:  since,
		Until:  until,
		Limit:  limit,
		Search: "random",
	}
	// Query events based on the filter
	var ch event.C
	go func() {
		ch, err = a.QueryEvents(f)
		if err != nil {
			fmt.Println("Failed to query events:", err)
			return
		}
		// close(ch)
	}()

	// Display queried events
	log.I.Ln("receiving events")
	for ev := range ch {
		fmt.Printf("ID: %s, Content: %s\n", ev.ID, ev.Content)
	}
}
