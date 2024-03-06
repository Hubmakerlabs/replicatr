package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"mleku.dev/git/nostr/context"

	"github.com/Hubmakerlabs/replicatr/pkg/ic/agent"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/keys"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/tag"
	"mleku.dev/git/nostr/timestamp"
)

func createRandomEvent(i int) (e event.T) {
	e = event.T{
		CreatedAt: timestamp.T(time.Now().Unix()),
		Kind:      kind.T(rand.Intn(500)),
		Tags:      []tag.T{{"tag1", "tag2"}, {"tag3"}},
		Content:   fmt.Sprintf("This is a random event content %d", i),
	}

	err := e.Sign(keys.GeneratePrivateKey())
	if err != nil {
		fmt.Println("unable to create random event number %d: %v", i, err)
	}

	return
}

func main() { //arg1 = portnum, arg2 = canisterID
	if len(os.Args) < 3 {
		fmt.Println("not enough args: 2 args required <canisterURL> <canisterID>")
	}
	// Initialize the agent with the configuration for a local replica
	a, err := agent.New(os.Args[2], os.Args[1])
	if err != nil {
		fmt.Printf("failed to initialize agent: %v\n", err)
	}

	var wg sync.WaitGroup

	// Create and save random events
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			event := createRandomEvent(i)
			err := a.SaveEvent(context.Bg(), &event)
			if err != nil {
				fmt.Printf("Failed to save event %d: %v\n", i, err)
				wg.Done()
				return
			}
			fmt.Printf("Event %d saved successfully\n", i)
			wg.Done()
		}(i)

	}

	wg.Wait()
	// Create a filter to query events
	// s := timestamp.Tp(time.Now().Add(-24 * time.Hour).Unix())
	// since := &s

	u := timestamp.Tp(time.Now().Unix())
	until := &u

	l := 10
	limit := &l

	f := &filter.T{
		Until:  until,
		Limit:  limit,
		Search: "random",
	}

	// Query events based on the filter
	ch := make(chan *event.T)
	go func() {
		err = a.QueryEvents(context.Bg(), ch, f)
		if err != nil {
			fmt.Println("Failed to query events:", err)
			return
		}
		close(ch)
	}()

	// Display queried events
	fmt.Println("Queried Events:")
	for ev := range ch {
		fmt.Printf("ID: %s, Content: %s\n", ev.ID, ev.Content)
	}
}
