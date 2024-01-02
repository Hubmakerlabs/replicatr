package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
)

type ProfileMetadata struct {
	// PubKey must always be set otherwise things will break
	PubKey string `json:"-"`
	// Event may be empty if a profile metadata event wasn't found
	Event *nip1.Event `json:"-,omitempty"`
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
	v, e := nip19.EncodePublicKey(p.PubKey)
	log.D.Chk(e)
	return v
}

func (p ProfileMetadata) Nprofile(ctx context.Context, sys *System,
	nrelays int) string {

	v, e := nip19.EncodeProfile(p.PubKey,
		sys.FetchOutboxRelays(ctx, p.PubKey))
	log.D.Chk(e)
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

func FetchProfileMetadata(ctx context.Context, pool *nostr.SimplePool,
	pubkey string, relays ...string) *ProfileMetadata {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch := pool.SubManyEose(ctx, relays, nip1.Filters{
		{
			Kinds:   kinds.T{kind.ProfileMetadata},
			Authors: []string{pubkey},
			Limit:   1,
		},
	})
	for ie := range ch {
		if m, err := ParseMetadata(ie.Event); err == nil {
			return m
		}
	}
	return &ProfileMetadata{PubKey: pubkey}
}

func ParseMetadata(ev *nip1.Event) (pm *ProfileMetadata, e error) {
	if ev.Kind != 0 {
		e = fmt.Errorf("event %s is kind %d, not 0", ev.ID, ev.Kind)
		return
	} else if e = json.Unmarshal([]byte(ev.Content), &pm); fails(e) {
		cont := ev.Content
		if len(cont) > 100 {
			cont = cont[0:99]
		}
		e = fmt.Errorf("failed to parse metadata (%s) from event %s: %w",
			cont, ev.ID, e)
		return
	}
	pm.PubKey = ev.PubKey
	pm.Event = ev
	return
}
