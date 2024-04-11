package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/fiatjaf/eventstore/badger"
	"github.com/gorilla/websocket"
	"github.com/nbd-wtf/go-nostr"
	// "github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
)

func FiltersTest(authors []string, ids []string, b *badger.BadgerBackend, numQueries int, seed *int, ctx context.T,
	c *websocket.Conn) error {

	// Construct query
	query := nostr.Filter{
		IDs:     ids,     // Filter by a subset of event IDs
		Authors: authors, // Filter by a subset of authors
	}

	// Query the relay
	queryResultRelay, err := queryRelay(c, query)
	if err != nil {
		return fmt.Errorf("Error querying relay: %v", err)
	}

	// Query the badger backend
	queryResultBadger, err := queryBadger(b, query, ctx)
	if err != nil {
		return fmt.Errorf("Error querying Badger backend: %v", err)
	}

	// Compare results (you'll likely want a more robust comparison than this)
	if !compareResults(queryResultBadger, queryResultRelay) {
		return fmt.Errorf("Query results mismatch")
	}

	fmt.Println("Filter Test Successful!")
	return nil
}

// Helper function to query the relay
func queryRelay(c *websocket.Conn, filter nostr.Filter) ([]nostr.Event, error) {
	// Generate a unique subscription ID
	subscriptionID := fmt.Sprintf("sub-%d", time.Now().UnixNano())

	// Send the REQ message
	query := []interface{}{"REQ", subscriptionID, filter}
	jsonData, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	err = c.WriteMessage(websocket.TextMessage, jsonData)
	if err != nil {
		return nil, err
	}

	// Event collection
	var events []nostr.Event

	// Read messages until EOSE is received
	for {
		counter := 0
		c.SetReadDeadline(time.Now().Add(10 * time.Second)) // Set a 10-second read deadline
		_, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				fmt.Printf("Connection closed normally.\n")
				break // Exit the loop if the connection is closed normally
			} else {
				return nil, fmt.Errorf("Failed to read relay response from event  or read timeout occurred: %v\n", err)
			}
		}

		var result []interface{}
		err = json.Unmarshal(message, &result)
		if err != nil {
			return nil, err
		}

		// Check message type
		if len(result) < 2 {
			return nil, fmt.Errorf("invalid message format: %v", result)
		}

		msgType, ok := result[0].(string)
		if !ok {
			return nil, fmt.Errorf("first element of message is not a string: %v", result)
		}

		switch msgType {
		case "EVENT":
			fmt.Printf("EVENT: counter at %d", counter)
			// Parse the event using EventEnvelope
			var eventEnv nostr.EventEnvelope
			err = eventEnv.UnmarshalJSON(message)
			if err != nil {
				return nil, fmt.Errorf("failed to parse event envelope: %v", err)
			}

			// Extract the event
			event := eventEnv.Event
			if err != nil {
				return nil, fmt.Errorf("failed to extract event from envelope: %v", err)
			}
			events = append(events, event)

		case "EOSE":
			fmt.Printf("EOSE: counter at %d", counter)
			// Check if the subscription ID matches
			if result[1].(string) != subscriptionID {
				return nil, fmt.Errorf("mismatched subscription ID in EOSE")
			}
			// Send the CLOSE message
			err = c.WriteMessage(websocket.TextMessage, []byte("[\"CLOSE\", \""+subscriptionID+"\"]"))
			if err != nil {
				return nil, err // Might want to handle this differently instead of a hard error
			}
			return events, nil
		case "CLOSED":
			fmt.Printf("CLOSED: counter at %d", counter)
			// Check if the subscription ID matches
			if result[1].(string) != subscriptionID {
				return nil, fmt.Errorf("mismatched subscription ID in CLOSED")
			}
			return events, nil

		default:
			return nil, fmt.Errorf("unknown message type: %s", msgType)
		}
	}
	return events, err
}

// Helper function to query Badger backend
func queryBadger(db *badger.BadgerBackend, filter nostr.Filter, ctx context.T) (events []nostr.Event, err error) {
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
		events = append(events, *event) // Dereference the pointer to store the value
	}

	return
}

// Helper function to compare results (may need refinement)
// Helper function to compare results
func compareResults(badgerEvents, relayEvents []nostr.Event) bool {
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
		return false
	}

	// Check if every ID in the Badger set exists in the relay set
	for id := range badgerIDSet {
		if _, exists := relayIDSet[id]; !exists {
			return false
		}
	}

	// If we reach here, all IDs match
	return true
}

// Helper to generate random tags
func generateRandomTags() nostr.TagMap {
	return nostr.TagMap{
		"e": {"randomId1", "randomId2"},
		"p": {"randomPubKey1"},
		// Add more tags as needed
	}
}
