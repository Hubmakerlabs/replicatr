package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"github.com/aviate-labs/agent-go"
	"github.com/aviate-labs/agent-go/candid/idl"
	"github.com/aviate-labs/agent-go/principal"
)

func createRandomEvent(i int) Event {
	return Event{
		ID:        fmt.Sprintf("eventID-%d", i),
		Pubkey:    fmt.Sprintf("pubkey-%d", i),
		CreatedAt: idl.NewInt(time.Now().Unix()),
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
	localReplicaURL, _ := url.Parse("http://localhost:46847")
	cfg := agent.Config{
		FetchRootKey: true,
		ClientConfig: &agent.ClientConfig{Host: localReplicaURL},
	}
	ag, err := agent.New(cfg)
	if err != nil {
		fmt.Println("Failed to create agent:", err)
		return
	}

	canisterID, err := principal.Decode("avqkn-guaaa-aaaaa-qaaea-cai")
	if err != nil {
		fmt.Printf("Unable to parse canisterID: %v\n", err)
	}

	// Create and save random events
	for i := 0; i < 5; i++ {
		event := createRandomEvent(i)
		_, err := SaveEvent(ag, canisterID, event)
		if err != nil {
			fmt.Printf("Failed to save event %d: %v\n", i, err)
			continue
		}
		fmt.Printf("Event %d saved successfully\n", i)
	}

	// Create a filter to query events
	filter := Filter{
		Since:  idl.NewInt(time.Now().Add(-24 * time.Hour).Unix()),
		Until:  idl.NewInt(time.Now().Unix()),
		Limit:  idl.NewInt(10),
		Search: "random",
	}

	// Query events based on the filter
	events, err := GetEvents(ag, canisterID, filter)
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
