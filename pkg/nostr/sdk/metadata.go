package sdk

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pool"
)

type ProfileMetadata struct {
	// PubKey must always be set otherwise things will break
	PubKey string `json:"-"`
	// Event may be empty if a profile metadata event wasn't found
	Event *event.T `json:"-,omitempty"`
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
	v, e := bech32encoding.EncodePublicKey(p.PubKey)
	log.D.Chk(e)
	return v
}

func (p ProfileMetadata) Nprofile(c context.T, sys *System,
	nrelays int) string {

	v, e := bech32encoding.EncodeProfile(p.PubKey,
		sys.FetchOutboxRelays(c, p.PubKey))
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

func FetchProfileMetadata(c context.T, pool *pool.SimplePool,
	pubkey string, relays ...string) (pm *ProfileMetadata) {

	c, cancel := context.Cancel(c)
	defer cancel()
	ch := pool.SubManyEose(c, relays, filters.T{
		{
			Kinds:   kinds.T{kind.ProfileMetadata},
			Authors: []string{pubkey},
			Limit:   1,
		},
	})
	var e error
	for ie := range ch {
		if pm, e = ParseMetadata(ie.T); !log.E.Chk(e) {
			return
		}
	}
	return &ProfileMetadata{PubKey: pubkey}
}

func ParseMetadata(ev *event.T) (pm *ProfileMetadata, e error) {
	if ev.Kind != 0 {
		e = fmt.Errorf("event %s is kind %d, not 0", ev.ID, ev.Kind)
		return
	} else if e = json.Unmarshal([]byte(ev.Content), &pm); log.Fail(e) {
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
