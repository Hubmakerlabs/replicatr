package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	filters2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	// connect to relay
	url := "wss://nostr.zebedee.cloud"
	rl, err := relays.RelayConnect(ctx, url)
	if err != nil {
		panic(err)
	}

	reader := os.Stdin
	var npub string
	var b [64]byte
	fmt.Fprintf(os.Stderr, "using %s\n----\nexample subscription for three most recent notes mentioning user\npaste npub key: ", url)
	if n, err := reader.Read(b[:]); err == nil {
		npub = strings.TrimSpace(fmt.Sprintf("%s", b[:n]))
	} else {
		panic(err)
	}

	// create filters
	var filters filters2.T
	if _, v, err := nip19.Decode(npub); err == nil {
		t := make(map[string][]string)
		// making a "p" tag for the above public key.
		// this filters for messages tagged with the user, mainly replies.
		t["p"] = []string{v.(string)}
		filters = []filter.T{{
			Kinds: []int{event.KindTextNote},
			Tags:  t,
			// limit = 3, get the three most recent notes
			Limit: 3,
		}}
	} else {
		panic("not a valid npub!")
	}

	// create a subscription and submit to relay
	// results will be returned on the sub.Events channel
	sub, _ := rl.Subscribe(ctx, filters)

	// we will append the returned events to this slice
	evs := make([]event.T, 0)

	go func() {
		<-sub.EndOfStoredEvents
		cancel()
	}()
	for ev := range sub.Events {
		evs = append(evs, *ev)
	}

	filename := "example_output.json"
	if f, err := os.Create(filename); err == nil {
		fmt.Fprintf(os.Stderr, "returned events saved to %s\n", filename)
		// encode the returned events in a file
		enc := json.NewEncoder(f)
		enc.SetIndent("", " ")
		enc.Encode(evs)
		f.Close()
	} else {
		panic(err)
	}

	fmt.Fprintf(os.Stderr, "----\nexample publication of note.\npaste nsec key (leave empty to autogenerate): ")
	var nsec string
	if n, err := reader.Read(b[:]); err == nil {
		nsec = strings.TrimSpace(fmt.Sprintf("%s", b[:n]))
	} else {
		panic(err)
	}

	var sk string
	ev := event.T{}
	if _, s, e := nip19.Decode(nsec); e == nil {
		sk = s.(string)
	} else {
		sk = keys.GeneratePrivateKey()
	}
	if pub, e := keys.GetPublicKey(sk); e == nil {
		ev.PubKey = pub
		if npub, e := nip19.EncodePublicKey(pub); e == nil {
			fmt.Fprintln(os.Stderr, "using:", npub)
		}
	} else {
		panic(e)
	}

	ev.CreatedAt = timestamp.Now()
	ev.Kind = event.KindTextNote
	var content string
	fmt.Fprintln(os.Stderr, "enter content of note, ending with an empty newline (ctrl+d):")
	for {
		if n, err := reader.Read(b[:]); err == nil {
			content = fmt.Sprintf("%s%s", content, fmt.Sprintf("%s", b[:n]))
		} else if err == io.EOF {
			break
		} else {
			panic(err)
		}
	}
	ev.Content = strings.TrimSpace(content)
	ev.Sign(sk)
	for _, url := range []string{"wss://nostr.zebedee.cloud"} {
		ctx := context.WithValue(context.Background(), "url", url)
		rl, e := relays.RelayConnect(ctx, url)
		if e != nil {
			fmt.Println(e)
			continue
		}
		fmt.Println("posting to: ", url)
		rl.Publish(ctx, ev)
	}
}
