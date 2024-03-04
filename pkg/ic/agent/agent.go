package agent

import (
	"fmt"
	"net/url"

	agent_go "github.com/aviate-labs/agent-go"
	"github.com/aviate-labs/agent-go/candid/idl"
	"github.com/aviate-labs/agent-go/principal"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/eventid"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/tags"
	"mleku.dev/git/nostr/timestamp"
)

const DefaultHost = "http://localhost:46847"

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

type Agent struct {
	Ag         *agent_go.Agent
	CanisterID principal.Principal
}

func (a *Agent) SaveCandidEvent(event Event) (string, error) {
	methodName := "save_event"
	args := []any{event}
	var result string
	err := a.Ag.Call(a.CanisterID, methodName, args, []any{&result})
	if err != nil {
		return "", err
	}

	if len(result) > 0 {
		return result, nil
	}

	return "", fmt.Errorf("unexpected result format")
}

func (a *Agent) GetCandidEvent(filter Filter) ([]Event, error) {
	methodName := "get_events"
	args := []any{filter}
	var result []Event

	err := a.Ag.Query(a.CanisterID, methodName, args, []any{&result})
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

func NewAgent(cid, canAddr string) (a *Agent, err error) {
	localReplicaURL, _ := url.Parse("http://" + canAddr)
	cfg := agent_go.Config{
		FetchRootKey: true,
		ClientConfig: &agent_go.ClientConfig{Host: localReplicaURL},
	}
	ag, err := agent_go.New(cfg)
	if err != nil {
		fmt.Println("Failed to create agent:", err)
		return nil, err
	}

	canisterId, _ := principal.Decode(cid)

	if err != nil {
		fmt.Printf("Unable to parse canisterID: %v\n", err)
	}

	return &Agent{ag, canisterId}, nil
}

func TagMaptoKV(t filter.TagMap) (keys []KeyValuePair) {
	keys = make([]KeyValuePair, 0, len(t))
	for k := range t {
		keys = append(keys, KeyValuePair{k, t[k]})
	}
	return
}
func FilterToCandid(f filter.T) (result Filter) {
	result = Filter{
		IDs:     f.IDs,
		Kinds:   f.Kinds.ToUint16(),
		Authors: f.Authors,
		Tags:    TagMaptoKV(f.Tags),
		Search:  f.Search,
	}
	if f.Since != nil {
		result.Since = idl.NewInt(f.Since.T().Int())
	}
	if f.Until != nil {
		result.Until = idl.NewInt(f.Until.T().Int())
	}
	if f.Limit != nil {
		result.Limit = idl.NewInt(*f.Limit)
	}

	return

}

func EventToCandid(e event.T) Event {

	return Event{
		e.ID.String(),
		e.PubKey,
		idl.NewInt(int(e.CreatedAt)),
		uint16(e.Kind),
		e.Tags.Slice(),
		e.Content,
		e.Sig,
	}
}

func CandidToEvent(e Event) event.T {
	var t tags.T
	for _, v := range e.Tags {
		t = append(t, v)
	}
	return event.T{
		ID:        eventid.T(e.ID),
		PubKey:    e.Pubkey,
		CreatedAt: timestamp.T(e.CreatedAt.BigInt().Int64()),
		Kind:      kind.T(e.Kind),
		Tags:      t,
		Content:   e.Content,
		Sig:       e.Sig,
	}
}

func (a *Agent) GetEvents(f filter.T) ([]event.T, error) {

	filter := FilterToCandid(f)

	candidEvents, err := a.GetCandidEvent(filter)
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

	return events, err

}

func (a *Agent) SaveEvent(e event.T) (string, error) {

	event := EventToCandid(e)

	return a.SaveCandidEvent(event)

}
