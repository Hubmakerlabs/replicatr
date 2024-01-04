package relay

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
)

const RELAY = "wss://nostr.mom"

// NOTE
//
// These tests require an internet connection and will fail if the above relay
// is offline or unreachable.

// test if we can fetch a couple of random events
func TestSubscribe(t *testing.T) {
	// log2.SetLogLevel(log2.Debug)
	rl := MustRelayConnect(RELAY)
	defer rl.Close()

	sub, e := rl.Subscribe(context.Background(),
		filters.T{
			{Kinds: kinds.T{kind.TextNote}, Limit: 2},
		})
	if e != nil {
		t.Errorf("subscription failed: %v", e)
		return
	}

	timeout := time.After(2 * time.Second)
	n := 0

out:
	for {
		select {
		case event := <-sub.Events:
			if event == nil {
				t.Errorf("event is nil: %v", event)
			}
			n++
		case <-sub.EndOfStoredEvents:
			break out
		case <-rl.Context().Done():
			t.Errorf("connection closed: %v", rl.Context().Err())
			break out
		case <-timeout:
			t.Errorf("timeout")
			break out
		}
	}
	if n != 2 {
		t.Errorf("expected 2 events, got %d", n)
	}
}

// test if we can do multiple nested subscriptions
func TestNestedSubscriptions(t *testing.T) {
	// log2.SetLogLevel(log2.Debug)
	rl := MustRelayConnect(RELAY)
	defer rl.Close()

	n := atomic.Uint32{}

	// fetch 2 replies to a note
	sub, e := rl.Subscribe(context.Background(), filters.T{{
		Kinds: kinds.T{kind.TextNote},
		Tags:  filter.TagMap{"e": []string{"0e34a74f8547e3b95d52a2543719b109fd0312aba144e2ef95cba043f42fe8c5"}},
		Limit: 3,
	}})
	if e != nil {
		t.Errorf("subscription 1 failed: %v", e)
		return
	}

	for {
		select {
		case event := <-sub.Events:
			// now fetch author of this
			sub, e := rl.Subscribe(context.Background(), filters.T{{Kinds: kinds.T{kind.SetMetadata}, Authors: []string{event.PubKey}, Limit: 1}})
			if e != nil {
				t.Errorf("subscription 2 failed: %v", e)
				return
			}

			for {
				select {
				case <-sub.Events:
					// do another subscription here in "sync" mode, just so we're sure things are not blocking
					evs, e := rl.QuerySync(context.Background(), &filter.T{Limit: 1})
					if fails(e) {

					}
					log.D.S(evs)
					n.Add(1)
					if n.Load() == 3 {
						// if we get here it means the test passed
						return
					}
				case <-sub.Context.Done():
					goto end
				case <-sub.EndOfStoredEvents:
					sub.Unsub()
				}
			}
		end:
			fmt.Println("")
		case <-sub.EndOfStoredEvents:
			sub.Unsub()
			return
		case <-sub.Context.Done():
			t.Errorf("connection closed: %v", rl.Context().Err())
			return
		}
	}
}
