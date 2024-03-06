package agent

import (
	"net/url"
	"os"

	agent_go "github.com/aviate-labs/agent-go"
	"github.com/aviate-labs/agent-go/candid/idl"
	"github.com/aviate-labs/agent-go/principal"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

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

type Backend struct {
	*agent_go.Agent
	CanisterID principal.Principal
}

func New(cid, canAddr string) (a *Backend, err error) {
	log.D.Ln("setting up IC backend to", canAddr, cid)
	a = &Backend{}
	localReplicaURL, _ := url.Parse("http://" + canAddr)
	cfg := agent_go.Config{
		FetchRootKey: true,
		ClientConfig: &agent_go.ClientConfig{Host: localReplicaURL},
	}
	if a.Agent, err = agent_go.New(cfg); chk.E(err) {
		return
	}
	if a.CanisterID, err = principal.Decode(cid); chk.E(err) {
		return
	}
	log.D.Ln("successfully connected to IC backend")
	return
}

func (a *Backend) SaveCandidEvent(event Event) (result string, err error) {
	methodName := "save_event"
	args := []any{event}
	if err = a.Call(a.CanisterID, methodName, args, []any{&result}); chk.E(err) {
		return
	}
	if len(result) > 0 {
		return
	}
	err = log.E.Err("unexpected result format")
	return
}

func (a *Backend) GetCandidEvent(filter *Filter) ([]Event, error) {
	methodName := "get_events"
	args := []any{*filter}
	log.I.S(filter)
	var result []Event
	err := a.Query(a.CanisterID, methodName, args, []any{&result})
	if err != nil {
		return nil, err
	}
	return result, err
}

func (a *Backend) QueryEvents(c context.T, ch chan *event.T, f *filter.T) (err error) {
	if f == nil {
		return log.E.Err("nil filter for query")
	}
	var candidEvents []Event
	if candidEvents, err = a.GetCandidEvent(FilterToCandid(f)); chk.E(err) {
		return
	}
	log.I.Ln("got", len(candidEvents), "events")
	for _, e := range candidEvents {
		// log.I.Ln("sending event", i)
		ch <- CandidToEvent(&e)
	}
	// log.I.Ln("done sending events")
	return
}

func (a *Backend) SaveEvent(c context.T, e *event.T) (err error) {
	var res string
	if res, err = a.SaveCandidEvent(EventToCandid(e)); chk.E(err) {
		return
	}
	if res != "success" {
		// this is unlikely to happen but since it could.
		err = log.E.Err("failed to store event", e.ToObject().String())
	}
	return
}
