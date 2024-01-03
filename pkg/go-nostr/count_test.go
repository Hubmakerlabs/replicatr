package nostr

import (
	"context"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
)

func TestCount(t *testing.T) {
	const RELAY = "wss://relay.nostr.band"

	rl := MustRelayConnect(RELAY)
	defer rl.Close()

	count, err := rl.Count(context.Background(), Filters{
		{Kinds: []int{event.KindContactList}, Tags: TagMap{"p": []string{"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"}}},
	})
	if err != nil {
		t.Errorf("count request failed: %v", err)
		return
	}

	if count <= 0 {
		t.Errorf("count result wrong: %v", count)
		return
	}
}
