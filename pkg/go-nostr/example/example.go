package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	filters2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
)

var log = log2.GetStd()

func main() {
	ctx, cancel := context.Timeout(context.Bg(), 3*time.Second)

	// connect to relay
	url := "wss://nostr.zebedee.cloud"
	rl, e := relays.RelayConnect(ctx, url)
	if e != nil {
		panic(e)
	}

	reader := os.Stdin
	var npub string
	var b [64]byte
	log.D.F("using %s\n----\nexample subscription for three most recent notes mentioning user\npaste npub key: ", url)
	if n, e := reader.Read(b[:]); e == nil {
		npub = strings.TrimSpace(fmt.Sprintf("%s", b[:n]))
	} else {
		panic(e)
	}

	// create filters
	var filters filters2.T
	if _, v, e := nip19.Decode(npub); e == nil {
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
	if f, e := os.Create(filename); e == nil {
		log.D.F("returned events saved to %s\n", filename)
		// encode the returned events in a file
		enc := json.NewEncoder(f)
		enc.SetIndent("", " ")
		enc.Encode(evs)
		f.Close()
	} else {
		panic(e)
	}

	log.D.F("----\nexample publication of note.\npaste nsec key (leave empty to autogenerate): ")
	var nsec string
	if n, e := reader.Read(b[:]); e == nil {
		nsec = strings.TrimSpace(fmt.Sprintf("%s", b[:n]))
	} else {
		panic(e)
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
		if n, e := reader.Read(b[:]); e == nil {
			content = fmt.Sprintf("%s%s", content, fmt.Sprintf("%s", b[:n]))
		} else if e == io.EOF {
			break
		} else {
			panic(e)
		}
	}
	ev.Content = strings.TrimSpace(content)
	ev.Sign(sk)
	for _, url := range []string{"wss://nostr.zebedee.cloud"} {
		ctx := context.Value(context.Bg(), "url", url)
		rl, e := relays.RelayConnect(ctx, url)
		if e != nil {
			fmt.Println(e)
			continue
		}
		fmt.Println("posting to: ", url)
		log.Fail(rl.Publish(ctx, &ev))
	}
}
