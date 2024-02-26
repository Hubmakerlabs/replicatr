package agent_nostricgo

import (
	"fmt"

	agent "github.com/aviate-labs/agent-go"
	"github.com/aviate-labs/agent-go/candid/idl"
	"github.com/aviate-labs/agent-go/principal"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
)

type KeyValuePair struct {
	Key   string   `json:"key"`
	Value []string `json:"value"`
}

type Event struct {
	ID        string     `json:"id"`
	Pubkey    string     `json:"pubkey"`
	CreatedAt idl.Int    `json:"created_at"`
	Kind      uint16     `json:"kind"`
	Tags      [][]string `json:"tags"`
	Content   string     `json:"content"`
	Sig       string     `json:"sig"`
}

type Filter struct {
	IDs     []string       `json:"ids"`
	Kinds   []uint16       `json:"kinds"`
	Authors []string       `json:"authors"`
	Tags    []KeyValuePair `json:"tags"`
	Since   idl.Int        `json:"since"`
	Until   idl.Int        `json:"until"`
	Limit   idl.Int        `json:"limit"`
	Search  string         `json:"search"`
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
	var result []any

	err := ag.Call(canisterID, methodName, args, &result)
	if err != nil {
		return "", err
	}

	// Assuming the result is a single text value
	if len(result) > 0 {
		if res, ok := result[0].(string); ok {
			return res, nil
		}
	}

	return "", fmt.Errorf("unexpected result format")
}

func GetEvents(ag *agent.Agent, canisterID principal.Principal, filter Filter) ([]Event, error) {
	methodName := "get_events"
	args := []any{filter}
	var result []Event

	err := ag.Query(canisterID, methodName, args, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func LocalConfigHelper() (*agent.Agent, principal.Principal) {
	identity := new(identity.AnonymousIdentity)

	// Parse the local replica URL
	localReplicaURL, _ := url.Parse("http://localhost:46847/")

	// Create a new agent configuration
	cfg := agent.Config{
		Identity:      identity,
		IngressExpiry: 5 * time.Minute,
		ClientConfig: &agent.ClientConfig{
			Host: localReplicaURL,
		},
	}

	// Initialize the agent with the configuration
	ag, err := agent.New(cfg)
	if err != nil {
		panic(err)
	}

	canisterID, _ := principal.Decode("testnet_backend")

	return ag, canisterID
}

func FilterToCandid(f filter.T)(Filter) {
	return Filter{
		f.IDs,
		f.Kinds,
		f.Authors,
		f.Tags,
		f.Since,
		f.Until,
		f.Limit,
		f.Search
}

}

func EventToCandid(e event.T)(Event) {
	return Event{
		e.ID,
		e.Pubkey,
		e.CreatedAt,
		e.Kind,
		e.Tags,
		e.Content,
		e.Sig
	}
}

func CandidToEvent(e Event)(event.T){
	return event.T{
		e.ID,
		e.Pubkey,
		e.CreatedAt,
		e.Kind,
		e.Tags,
		e.Content,
		e.Sig
	}
}

func LocalGetEvents(f filter.T)([]event.T, error) {
	ag, canisterID := LocalConfigHelper()
	
	filter := FilterToCandid(f)

	candidEvents, err := myiclib.GetEvents(ag, canisterID, filter)
    if err != nil {
        fmt.Println("Error getting events:", err)
        return nil, err
    }

	 var events []event.T

	 // Iterate through the slice of Person structs
	 for _, e := range candidEvents {
		 // Add the Name field of each Person to the names slice
		events = append(events, CandidToEvent(e))
	 }

	 return events,err

}

func LocalSaveEvent(e event.T ) (string, error) {
	ag, canisterID := LocalConfigHelper()
	event := EventToCandid(e)
	

	return SaveEvent(ag, canisterID, event)
    
}
