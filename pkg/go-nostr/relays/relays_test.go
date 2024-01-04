package relays

import (
	"context"
	"testing"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
)

func TestEOSEMadness(t *testing.T) {
	rl := MustRelayConnect(eose.RELAY)
	defer rl.Close()

	sub, err := rl.Subscribe(context.Background(), filters.T{
		{Kinds: []int{event.KindTextNote}, Limit: 2},
	})
	if err != nil {
		t.Errorf("subscription failed: %v", err)
		return
	}

	timeout := time.After(3 * time.Second)
	n := 0
	e := 0

	for {
		select {
		case event := <-sub.Events:
			if event == nil {
				t.Fatalf("event is nil: %v", event)
			}
			n++
		case <-sub.EndOfStoredEvents:
			e++
			if e > 1 {
				t.Fatalf("eose infinite loop")
			}
			continue
		case <-rl.Context().Done():
			t.Fatalf("connection closed: %v", rl.Context().Err())
		case <-timeout:
			goto end
		}
	}

end:
	if e != 1 {
		t.Fatalf("didn't get an eose")
	}
	if n < 2 {
		t.Fatalf("didn't get events")
	}
}

func TestCount(t *testing.T) {
	const RELAY = "wss://relay.nostr.band"

	rl := MustRelayConnect(RELAY)
	defer rl.Close()

	count, err := rl.Count(context.Background(), filters.T{
		{Kinds: []int{event.KindContactList}, Tags: filter.TagMap{"p": []string{"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"}}},
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
