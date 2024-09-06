package client

// func TestEOSEMadness(t *testing.T) {
// 	rl := MustConnect(eoseenvelope.RELAY)
// 	defer chk.D(rl.Close())
//
// 	sub, e := rl.Subscribe(context.Bg(), filters.T{
// 		{Kinds: kinds.T{kind.TextNote}, Limit: 2},
// 	})
// 	if e != nil {
// 		t.Errorf("subscription failed: %v", e)
// 		return
// 	}
//
// 	timeout := time.After(3 * time.Second)
// 	n := 0
// 	ee := 0
//
// 	for {
// 		select {
// 		case ev := <-sub.Events:
// 			if ev == nil {
// 				t.Fatalf("event is nil: %v", ev)
// 			}
// 			n++
// 		case <-sub.EndOfStoredEvents:
// 			ee++
// 			if ee > 1 {
// 				t.Fatalf("eose infinite loop")
// 			}
// 			continue
// 		case <-rl.Context().Done():
// 			t.Fatalf("connection closed: %v", rl.Context().Err())
// 		case <-timeout:
// 			goto end
// 		}
// 	}
//
// end:
// 	if ee != 1 {
// 		t.Fatalf("didn't get an eose")
// 	}
// 	if n < 2 {
// 		t.Fatalf("didn't get events")
// 	}
// }

// func TestCount(t *testing.T) {
// 	const RELAY = "wss://relay.nostr.band"
//
// 	rl := MustConnect(RELAY)
// 	defer chk.D(rl.Close())
//
// 	count, e := rl.Count(context.Bg(), filters.T{
// 		{Kinds: kinds.T{kind.FollowList}, Tags: filter.TagMap{"p": []string{"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"}}},
// 	})
// 	if e != nil {
// 		t.Errorf("count request failed: %v", e)
// 		return
// 	}
//
// 	if count <= 0 {
// 		t.Errorf("count result wrong: %v", count)
// 		return
// 	}
// }
