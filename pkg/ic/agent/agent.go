package agent

import (
	"net/url"
	"os"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	agent_go "github.com/aviate-labs/agent-go"
	"github.com/aviate-labs/agent-go/principal"
	"mleku.dev/git/atomic"
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
	CreatedAt int64      `ic:"created_at"`
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
	Since   int64          `ic:"since"`
	Until   int64          `ic:"until"`
	Limit   int64          `ic:"limit"`
	Search  string         `ic:"search"`
}

type Backend struct {
	Ctx        context.T
	Agent      *agent_go.Agent
	CanisterID principal.Principal
}

func New(c context.T, cid, canAddr string) (a *Backend, err error) {
	log.D.Ln("setting up IC backend to", canAddr, cid)
	a = &Backend{Ctx: c}
	localReplicaURL, _ := url.Parse(canAddr)
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

func (b *Backend) SaveCandidEvent(event Event) (result string, err error) {
	methodName := "save_event"
	args := []any{event}
	if err = b.Agent.Call(b.CanisterID, methodName, args,
		[]any{&result}); chk.E(err) {
		return
	}
	if len(result) > 0 {
		return
	}
	err = log.E.Err("unexpected result format")
	return
}

func (b *Backend) GetCandidEvent(c context.T, filter *Filter) ([]Event, error) {
	methodName := "get_events"
	args := []any{*filter}
	// log.T.S(filter)
	var result []Event
	err := b.Agent.Query(b.CanisterID, methodName, args, []any{&result})
	if err != nil {
		return nil, err
	}
	return result, err
}

func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.C, err error) {

	if f == nil {
		err = log.E.Err("nil filter for query")
		return
	}
	var candidEvents []Event
	if candidEvents, err = b.GetCandidEvent(c, FilterToCandid(f)); chk.E(err) {
		return
	}
	log.I.Ln("got", len(candidEvents), "events")
	for i, e := range candidEvents {
		select {
		case <-b.Ctx.Done():
			return
		default:
		}
		log.I.Ln("sending event", i)
		ch <- CandidToEvent(&e)
	}
	log.I.Ln("done sending events")
	return
}

func (b *Backend) SaveEvent(c context.T, e *event.T) (err error) {
	select {
	case <-c.Done():
		return
	default:
	}
	var res string
	if res, err = b.SaveCandidEvent(EventToCandid(e)); chk.E(err) {
		return
	}
	if res != "success" {
		// this is unlikely to happen but since it could.
		err = log.E.Err("failed to store event", e.ToObject().String())
	}
	return
}

// DeleteEvent deletes an event matching the given event.
// todo: not yet implemented, but there is already a backend function for this
func (b *Backend) DeleteEvent(c context.T, ev *event.T) (err error) {
	log.W.Ln("delete events on IC not yet implemented")
	// todo: if event is not found, return eventstore.ErrEventNotExists
	return
}

// CountEvents counts how many events match the filter in the IC.
// todo: use the proper count events API call in the canister
func (b *Backend) CountEvents(c context.T, f *filter.T) (count int, err error) {
	ch := make(chan *event.T)
	done := make(chan struct{})
	var counter atomic.Int32
	go func(ch chan *event.T) {
		// receive events and increment counter
		for _ = range ch {
			counter.Add(1)
		}
		close(done)
	}(ch)
	if ch, err = b.QueryEvents(c, f); chk.E(err) {
		return
	}
out:
	for {
		select {
		case <-c.Done():
			break out
		// todo: there should be some place to set this timeout more centrally
		case <-time.After(time.Second * 5):
			break out
		case <-done:
			break out
		}
	}
	count = int(counter.Load())
	return
}

func (b *Backend) ClearEvents(c context.T) (result string, err error) {
	methodName := "clear_events"
	args := []any{}
	if err = b.Agent.Call(b.CanisterID, methodName, args, []any{&result}); chk.E(err) {
		return "", err
	}

	if len(result) == 0 {
		return "", log.E.Err("unexpected result format from clear_events")
	}

	return result, nil
}
