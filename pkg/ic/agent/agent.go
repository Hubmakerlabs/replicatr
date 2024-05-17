package agent

import (
	"net/url"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	agent_go "github.com/aviate-labs/agent-go"
	"github.com/aviate-labs/agent-go/identity"
	"github.com/aviate-labs/agent-go/principal"

	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	sec "github.com/aviate-labs/secp256k1"
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

func New(c context.T, cid, canAddr, secKey string) (a *Backend, err error) {
	log.D.Ln("setting up IC backend to", canAddr, cid)
	a = &Backend{Ctx: c}
	localReplicaURL, _ := url.Parse(canAddr)
	secKeyBytes, err := hex.Dec(secKey)
	if err != nil {
		return nil, err
	}
	privKey, _ := sec.PrivKeyFromBytes(sec.S256(), secKeyBytes)
	id, _ := identity.NewSecp256k1Identity(privKey)
	cfg := agent_go.Config{
		FetchRootKey: true,
		ClientConfig: &agent_go.ClientConfig{Host: localReplicaURL},
		Identity:     id,
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

func (b *Backend) SaveCandidEvent(event Event) (err error) {

	methodName := "save_event"
	var result *string
	args := []any{event, time.Now().UnixNano()}
	err = b.Agent.Call(b.CanisterID, methodName, args,
		[]any{&result})
	if err == nil && result != nil {
		err = log.E.Err("Unable to Store Event")
	}
	return
}

func (b *Backend) DeleteCandidEvent(event Event) (err error) {
	methodName := "delete_event"
	args := []any{event, time.Now().UnixNano()}
	var result *string
	err = b.Agent.Call(b.CanisterID, methodName, args,
		[]any{&result})
	if err == nil && result != nil {
		err = log.E.Err("Unable to Delete Event")
	}
	return
}

func (b *Backend) GetCandidEvent(filter *Filter) ([]Event, error) {
	methodName := "get_events"
	args := []any{*filter, time.Now().UnixNano()}
	var result []Event
	err := b.Agent.Query(b.CanisterID, methodName, args, []any{&result})
	if err != nil {
		return nil, err
	}
	return result, err
}

func (b *Backend) CountCandidEvent(filter *Filter) (int, error) {
	methodName := "count_events"
	args := []any{*filter, time.Now().UnixNano()}
	var result int
	err := b.Agent.Query(b.CanisterID, methodName, args, []any{&result})
	if err != nil {
		return -1, err
	}
	return result, err
}

func (b *Backend) ClearCandidEvents() (err error) {
	methodName := "clear_events"
	var result *string
	args := []any{time.Now().UnixNano()}
	err = b.Agent.Call(b.CanisterID, methodName, args, []any{&result})

	if err == nil && result != nil {
		err = log.E.Err("Unable to Clear Events")
	}

	return
}
