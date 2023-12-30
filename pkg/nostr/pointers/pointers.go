package pointers

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
)

type Profile struct {
	PublicKey string   `json:"pubkey"`
	Relays    []string `json:"relays,omitempty"`
}

type Event struct {
	ID     nip1.EventID `json:"id"`
	Relays []string     `json:"relays,omitempty"`
	Author string       `json:"author,omitempty"`
	Kind   kind.T       `json:"kind,omitempty"`
}

type Entity struct {
	PublicKey  string   `json:"pubkey"`
	Kind       kind.T   `json:"kind,omitempty"`
	Identifier string   `json:"identifier,omitempty"`
	Relays     []string `json:"relays,omitempty"`
}
