package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"time"

	myiclib "agent_nostricgo"

	"github.com/aviate-labs/agent-go"
	"github.com/aviate-labs/agent-go/identity"
	"github.com/aviate-labs/agent-go/principal"
)

func createRandomEvent(i int) myiclib.Event {
	return myiclib.Event{
		ID:        fmt.Sprintf("eventID-%d", i),
		Pubkey:    fmt.Sprintf("pubkey-%d", i),
		CreatedAt: myiclib.Int(time.Now().Unix()),
		Kind:      uint16(rand.Intn(5)),
		Tags:      [][]string{{"tag1", "tag2"}, {"tag3"}},
		Content:   fmt.Sprintf("This is a random event content %d", i),
		Sig:       fmt.Sprintf("signature-%d", i),
	}
}

func main() {
	// Setup randomness
	rand.Seed(time.Now().UnixNano())

	// Initialize the agent with the configuration for a local replica
	identity := new(identity.AnonymousIdentity)
	localReplicaURL, _ := url.Parse("http://localhost:46847/")
	cfg := agent.Config{
		Identity:      identity,
		IngressExpiry: 5 * time.Minute,
		ClientConfig: &agent.ClientConfig{
			Host: localReplicaURL,
		},
	}
	ag, err := agent.New(cfg)
	if err != nil {
		fmt.Println("Failed to create agent:", err)
		return
	}

	// Assume canisterID is known and parsed correctly
	canisterID, _ := principal.Decode("your-canister-id")

	// Create and save random events
	for i := 0; i < 5; i++ {
		event := createRandomEvent(i)
		_, err := myiclib.SaveEvent(ag, canisterID, event)
		if err != nil {
			fmt.Printf("Failed to save event %d: %v\n", i, err)
			continue
		}
		fmt.Printf("Event %d saved successfully\n", i)
	}

	// Create a filter to query events
	filter := myiclib.Filter{
		Since:  myiclib.Int(time.Now().Add(-24 * time.Hour).Unix()), // events from the last 24 hours
		Until:  myiclib.Int(time.Now().Unix()),
		Limit:  myiclib.Int(10),
		Search: "random", // adjust based on your events' content or other attributes
	}

	// Query events based on the filter
	events, err := myiclib.GetEvents(ag, canisterID, filter)
	if err != nil {
		fmt.Println("Failed to query events:", err)
		return
	}

	// Display queried events
	fmt.Println("Queried Events:")
	for _, event := range events {
		fmt.Printf("ID: %s, Content: %s\n", event.ID, event.Content)
	}
}
