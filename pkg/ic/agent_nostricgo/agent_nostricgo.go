package main

import (
	"fmt"

	agent "github.com/aviate-labs/agent-go"
	"github.com/aviate-labs/agent-go/candid/idl"
	"github.com/aviate-labs/agent-go/principal"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
)

type KeyValuePair struct {
	Key   string   `ic:"key"`
	Value []string `ic:"value"`
}

type Event struct {
	ID        string     `ic:"id"`
	Pubkey    string     `ic:"pubkey"`
	CreatedAt idl.Int    `ic:"created_at"`
	Kind      uint16     `ic:"kind"`
	Tags      [][]string `ic:"tags"`
	Content   string     `ic:"content"`
	Sig       string     `ic:"sig"`
}

type Filter struct {
	IDs     []string       `ic:"ids"`
	Kinds   []uint16       `ic:"kinds"`
	Authors []string       `ic:"authors"`
	Tags    []KeyValuePair `ic:"tags"`
	Since   idl.Int        `ic:"since"`
	Until   idl.Int        `ic:"until"`
	Limit   idl.Int        `ic:"limit"`
	Search  string         `ic:"search"`
}

type E struct {
	Event event.T
}

type F struct {
	Filter filter.T
}

func SaveEvent(ag *agent.Agent, canisterID principal.Principal, event Event) (string, error) {
	methodName := "save_event"
	args := []any{event}
	var result string
	err := ag.Call(canisterID, methodName, args, []any{&result})
	if err != nil {
		return "", err
	}

	if len(result) > 0 {
		return result, nil
	}

	return "", fmt.Errorf("unexpected result format")
}

func GetEvents(ag *agent.Agent, canisterID principal.Principal, filter Filter) ([]Event, error) {
	methodName := "get_events"
	args := []any{filter}
	var result []Event

	err := ag.Query(canisterID, methodName, args, []any{&result})
	if err != nil {
		return nil, err
	}

	return result, err
}

func mapToEvent(item map[string]interface{}) (Event, error) {

	event := Event{
		ID: item["id"].(string),
	}

	return event, nil
}

// func LocalConfigHelper() (*agent.Agent, principal.Principal) {
// 	identity := new(identity.AnonymousIdentity)

// 	// Parse the local replica URL
// 	localReplicaURL, _ := url.Parse("http://localhost:46847/")

// 	// Create a new agent configuration
// 	cfg := agent.Config{
// 		Identity:      identity,
// 		IngressExpiry: 5 * time.Minute,
// 		ClientConfig: &agent.ClientConfig{
// 			Host: localReplicaURL,
// 		},
// 	}

// 	// Initialize the agent with the configuration
// 	ag, err := agent.New(cfg)
// 	if err != nil {
// 		panic(err)
// 	}

// 	canisterID, _ := principal.Decode("testnet_backend")

// 	return ag, canisterID
// }

// func FilterToCandid(f filter.T)(Filter) {
// 	return Filter{
// 		f.IDs,
// 		f.Kinds,
// 		f.Authors,
// 		f.Tags,
// 		f.Since,
// 		f.Until,
// 		f.Limit,
// 		f.Search,
// }

// }

// func EventToCandid(e event.T)(Event) {
// 	return Event{
// 		e.ID,
// 		e.Pubkey,
// 		e.CreatedAt,
// 		e.Kind,
// 		e.Tags,
// 		e.Content,
// 		e.Sig,
// 	}
// }

// func CandidToEvent(e Event)(event.T){
// 	return event.T{
// 		ID: e.ID,
// 		PubKey: e.Pubkey,
// 		CreatedAt: e.CreatedAt,
// 		Kind: e.Kind,
// 		Tags: e.Tags,
// 		Content: e.Content,
// 		Sig: e.Sig,
// 	}
// }

// func LocalGetEvents(f filter.T)([]event.T, error) {
// 	ag, canisterID := LocalConfigHelper()

// 	filter := FilterToCandid(f)

// 	candidEvents, err := GetEvents(ag, canisterID, filter)
//     if err != nil {
//         fmt.Println("Error getting events:", err)
//         return nil, err
//     }

// 	 var events []event.T

// 	 // Iterate through the slice of Person structs
// 	 for _, e := range candidEvents {
// 		 // Add the Name field of each Person to the names slice
// 		events = append(events, CandidToEvent(e))
// 	 }

// 	 return events,err

// }

// func LocalSaveEvent(e event.T ) (string, error) {
// 	ag, canisterID := LocalConfigHelper()
// 	event := EventToCandid(e)

// 	return SaveEvent(ag, canisterID, event)

// }
