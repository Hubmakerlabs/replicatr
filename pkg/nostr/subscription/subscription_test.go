package subscription_test

// const RELAY = "wss://nos.lol"

// test if we can fetch a couple of random events
// func TestSubscribeBasic(t *testing.T) {
// 	rl := relay.MustConnect(RELAY)
// 	defer rl.Close()
//
// 	sub, e := rl.Subscribe(context.Bg(), filters.T{{
// 		Kinds: kinds.T{kind.TextNote},
// 		Limit: 2},
// 	})
// 	if e != nil {
// 		t.Fatalf("subscription failed: %v", e)
// 		return
// 	}
//
// 	timeout := time.After(5 * time.Second)
// 	n := 0
//
// 	for {
// 		select {
// 		case event := <-sub.Events:
// 			if event == nil {
// 				t.Fatalf("event is nil: %v", event)
// 			}
// 			n++
// 		case <-sub.EndOfStoredEvents:
// 			goto end
// 		case <-rl.Context().Done():
// 			t.Fatalf("connection closed: %v", rl.Context().Err())
// 			goto end
// 		case <-timeout:
// 			t.Fatalf("timeout")
// 			goto end
// 		}
// 	}
//
// end:
// 	if n != 2 {
// 		t.Fatalf("expected 2 events, got %d", n)
// 	}
// }

// test if we can do multiple nested subscriptions
// func TestNestedSubscriptions(t *testing.T) {
// 	rl := relay.MustConnect(RELAY)
// 	defer rl.Close()
//
// 	n := atomic.Uint32{}
//
// 	// fetch 2 replies to a note
// 	sub, e := rl.Subscribe(context.Bg(), filters.T{{
// 		Kinds: kinds.T{kind.TextNote},
// 		Tags: filter.TagMap{
// 			"e": []string{
// 				"0e34a74f8547e3b95d52a2543719b109fd0312aba144e2ef95cba043f42fe8c5"},
// 		},
// 		Limit: 3}})
// 	if e != nil {
// 		t.Fatalf("subscription 1 failed: %v", e)
// 		return
// 	}
//
// 	for {
// 		select {
// 		case evt := <-sub.Events:
// 			// now fetch author of this
// 			sub, e := rl.Subscribe(context.Bg(), filters.T{{
// 				Kinds:   kinds.T{kind.ProfileMetadata},
// 				Authors: []string{evt.PubKey},
// 				Limit:   1}})
// 			if e != nil {
// 				t.Fatalf("subscription 2 failed: %v", e)
// 				return
// 			}
//
// 			for {
// 				select {
// 				case <-sub.Events:
// 					// do another subscription here in "sync" mode, just so
// 					// we're sure things are not blocking
// 					_, _ = rl.QuerySync(context.Bg(), &filter.T{Limit: 1})
//
// 					n.Add(1)
// 					if n.Load() == 3 {
// 						// if we get here it means the test passed
// 						return
// 					}
// 				case <-sub.Context.Done():
// 					goto end
// 				case <-sub.EndOfStoredEvents:
// 					sub.Unsub()
// 				}
// 			}
// 		end:
// 			fmt.Println("")
// 		case <-sub.EndOfStoredEvents:
// 			sub.Unsub()
// 			return
// 		case <-sub.Context.Done():
// 			t.Fatalf("connection closed: %v", rl.Context().Err())
// 			return
// 		}
// 	}
// }
