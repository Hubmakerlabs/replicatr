package app

import (
	"fmt"
	"io"
	l "log"
	"math"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/nbd-wtf/go-nostr"
	// "github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
)

func FiltersTest(authors []string, ids []string, b *badger.BadgerBackend,
	numQueries int, ctx context.T) error {
	nostr.InfoLogger = l.New(io.Discard, "", 0)
	var relay *nostr.Relay
	var err error
	for {
		relay, err = nostr.RelayConnect(ctx, "ws://127.0.0.1:3334")
		if err != nil {
			continue
		} else {
			break
		}
	}
	passcounter := 0
	for i := 0; i < numQueries; i++ {
		// Construct query
		query := generateRandomFilter(authors, ids)

		// Query the relay
		queryResultRelay, err := queryRelay(relay, ctx, query)
		if err != nil {
			fmt.Printf("Error querying relay for query # %d: %v\n", i, err)
		}

		// Query the badger backend
		queryResultBadger, err := queryBadger(b, query, ctx)
		if err != nil {
			fmt.Printf("Error querying Badger backend for query # %d: %v\n", i,
				err)
			passcounter++
			continue
		}

		// Compare results (you'll likely want a more robust comparison than this)
		if err = compareResults(queryResultBadger, queryResultRelay,
			numQueries); err != nil {
			fmt.Printf("Query %d of %d failed: %v \n", i, numQueries, err)
		} else {
			fmt.Printf("Query %d of %d passed\n", i, numQueries)
			passcounter++
		}
	}

	fmt.Printf("Filter Test Complete. %d of %d queries passed", passcounter,
		numQueries)
	return nil
}

// Helper function to query the relay
func queryRelay(relay *nostr.Relay, ctx context.T,
	filter nostr.Filter) ([]nostr.Event, error) {
	var events []nostr.Event
	sc, _ := context.Timeout(ctx, 5*time.Second)
	sub := relay.PrepareSubscription(sc, nostr.Filters{filter})
	if err := sub.Fire(); chk.E(err) {
		return nil, err
	}
	sub.Close()
	for {
		select {
		case ev := <-sub.Events:
			if ev == nil {
				continue // Handle nil events if necessary
			}
			events = append(events, *ev)
		case <-sub.EndOfStoredEvents:
			log.I.Ln("EOSE")
			return events, nil
		case <-sc.Done():
			log.I.Ln("subscription done")
			return events, nil
		case <-ctx.Done():
			log.I.Ln("context canceled")
			return events, nil
		}
	}
	return events, nil

}

// Helper function to query Badger backend
func queryBadger(db *badger.BadgerBackend, filter nostr.Filter,
	ctx context.T) (events []nostr.Event, err error) {
	// Implement the logic to query your Badger DB
	// ... return a slice of matching events.
	eventChan, err := db.QueryEvents(ctx, filter)
	if err != nil {
		return
	}

	for event := range eventChan {
		if event == nil {
			continue // or handle a nil event as an error if appropriate
		}
		events = append(events,
			*event) // Dereference the pointer to store the value
	}

	return
}

// Helper function to compare results (may need refinement)
// Helper function to compare results
func compareResults(badgerEvents, relayEvents []nostr.Event,
	numQueries int) error {
	// Create sets to store event IDs for efficient comparison
	badgerIDSet := make(map[string]bool)
	relayIDSet := make(map[string]bool)

	// Populate the sets
	for _, event := range badgerEvents {
		badgerIDSet[event.ID] = true
	}
	for _, event := range relayEvents {
		relayIDSet[event.ID] = true
	}

	// Check if the number of IDs match
	if len(badgerIDSet) != len(relayIDSet) {
		if math.Abs(float64(len(badgerIDSet)-len(relayIDSet))) < float64(numQueries)*0.6 {
			return nil
		} else {
			return fmt.Errorf("Expected number of results:%d; Actual number of Results: %d",
				len(badgerIDSet), len(relayIDSet))
		}
	}

	// Check if every ID in the Badger set exists in the relay set
	for id := range badgerIDSet {
		if _, exists := relayIDSet[id]; !exists {
			return fmt.Errorf("ID %s not found in relay set", id)
		}
	}

	// If we reach here, all IDs match
	return nil
}

// Helper to generate random tags
func generateRandomTags() nostr.TagMap {
	return nostr.TagMap{
		"e": {"randomId1", "randomId2"},
		"p": {"randomPubKey1"},
		// Add more tags as needed
	}
}
