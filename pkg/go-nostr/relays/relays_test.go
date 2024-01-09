package relays

import (
	"testing"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
)

func TestEOSEMadness(t *testing.T) {
	rl := MustRelayConnect(eose.RELAY)
	defer rl.Close()

	sub, e := rl.Subscribe(context.Bg(), filters.T{
		{Kinds: []int{event.KindTextNote}, Limit: 2},
	})
	if e != nil {
		t.Errorf("subscription failed: %v", e)
		return
	}

	timeout := time.After(3 * time.Second)
	n := 0
	ee := 0

	for {
		select {
		case ev := <-sub.Events:
			if ev == nil {
				t.Fatalf("event is nil: %v", ev)
			}
			n++
		case <-sub.EndOfStoredEvents:
			ee++
			if ee > 1 {
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
	if ee != 1 {
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

	count, e := rl.Count(context.Bg(), filters.T{
		{Kinds: []int{event.KindContactList}, Tags: filter.TagMap{"p": []string{"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"}}},
	})
	if e != nil {
		t.Errorf("count request failed: %v", e)
		return
	}

	if count <= 0 {
		t.Errorf("count result wrong: %v", count)
		return
	}
}
