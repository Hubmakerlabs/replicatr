package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/pools"
)

type ProfileMetadata struct {
	PubKey string   `json:"-"` // must always be set otherwise things will break
	Event  *event.T `json:"-"` // may be empty if a profile metadata event wasn't found

	// every one of these may be empty
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	About       string `json:"about,omitempty"`
	Website     string `json:"website,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Banner      string `json:"banner,omitempty"`
	NIP05       string `json:"nip05,omitempty"`
	LUD16       string `json:"lud16,omitempty"`
}

func (p ProfileMetadata) Npub() string {
	v, _ := nip19.EncodePublicKey(p.PubKey)
	return v
}

func (p ProfileMetadata) Nprofile(ctx context.Context, sys *System, nrelays int) string {
	v, _ := nip19.EncodeProfile(p.PubKey, sys.FetchOutboxRelays(ctx, p.PubKey))
	return v
}

func (p ProfileMetadata) ShortName() string {
	if p.Name != "" {
		return p.Name
	}
	if p.DisplayName != "" {
		return p.DisplayName
	}
	npub := p.Npub()
	return npub[0:7] + "…" + npub[58:]
}

func FetchProfileMetadata(ctx context.Context, pool *pools.SimplePool, pubkey string, relays ...string) ProfileMetadata {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := pool.SubManyEose(ctx, relays, filters.T{
		{
			Kinds:   []int{event.KindProfileMetadata},
			Authors: []string{pubkey},
			Limit:   1,
		},
	})

	for ie := range ch {
		if m, e := ParseMetadata(ie.T); e == nil {
			return m
		}
	}

	return ProfileMetadata{PubKey: pubkey}
}

func ParseMetadata(event *event.T) (meta ProfileMetadata, e error) {
	if event.Kind != 0 {
		e = fmt.Errorf("event %s is kind %d, not 0", event.ID, event.Kind)
	} else if e := json.Unmarshal([]byte(event.Content), &meta); e != nil {
		cont := event.Content
		if len(cont) > 100 {
			cont = cont[0:99]
		}
		e = fmt.Errorf("failed to parse metadata (%s) from event %s: %w", cont, event.ID, e)
	}

	meta.PubKey = event.PubKey
	meta.Event = event
	return meta, e
}
