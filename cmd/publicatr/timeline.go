package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip4"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/sdk"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func doDMList(cCtx *cli.Context) error {
	j := cCtx.Bool("json")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	// get followers
	followsMap, err := cfg.GetFollows(cCtx.String("a"))
	if err != nil {
		return err
	}

	var sk string
	var npub string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	if npub, err = nip19.GetPublicKey(sk); err != nil {
		return err
	}

	// get timeline
	filter := &nip1.Filter{
		Kinds:   kinds.T{kind.EncryptedDirectMessage},
		Authors: []string{npub},
	}

	evs := cfg.Events(filter)
	type entry struct {
		name   string
		pubkey string
	}
	var users []entry
	m := map[string]struct{}{}
	for _, ev := range evs {
		p := ev.Tags.GetFirst([]string{"p"}).Value()
		if _, ok := m[p]; ok {
			continue
		}
		if profile, ok := followsMap[p]; ok {
			m[p] = struct{}{}
			p, _ = nip19.EncodePublicKey(p)
			users = append(users, entry{
				name:   profile.DisplayName,
				pubkey: p,
			})
		} else {
			users = append(users, entry{
				name:   p,
				pubkey: p,
			})
		}
	}
	if j {
		for _, user := range users {
			log.D.Chk(json.NewEncoder(os.Stdout).Encode(user))
		}
		return nil
	}
	for _, user := range users {
		color.Set(color.FgHiRed)
		fmt.Print(user.name)
		color.Set(color.Reset)
		fmt.Print(": ")
		color.Set(color.FgHiBlue)
		fmt.Println(user.pubkey)
		color.Set(color.Reset)
	}
	return nil
}

func doDMTimeline(cCtx *cli.Context) error {
	u := cCtx.String("u")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	var sk string
	var npub string
	var err error
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	if npub, err = nip19.GetPublicKey(sk); err != nil {
		return err
	}

	if u == "me" {
		u = npub
	}
	var pub string
	if pp := sdk.InputToProfile(context.TODO(), u); pp != nil {
		pub = pp.PublicKey
	} else {
		return fmt.Errorf("failed to parse pubkey from '%s'", u)
	}
	// get followers
	followsMap, err := cfg.GetFollows(cCtx.String("a"))
	if err != nil {
		return err
	}

	// get timeline
	filter := &nip1.Filter{
		Kinds:   kinds.T{kind.EncryptedDirectMessage},
		Authors: []string{npub, pub},
		Tags:    nip1.TagMap{"p": []string{npub, pub}},
		Limit:   9999,
	}

	evs := cfg.Events(filter)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}

func doDMPost(cCtx *cli.Context) error {
	u := cCtx.String("u")
	stdin := cCtx.Bool("stdin")
	if !stdin && cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	sensitive := cCtx.String("sensitive")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	ev := &nip1.Event{}
	if npub, err := nip19.GetPublicKey(sk); err == nil {
		if _, err := nip19.EncodePublicKey(npub); err != nil {
			return err
		}
		ev.PubKey = npub
	} else {
		return err
	}

	if stdin {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		ev.Content = string(b)
	} else {
		ev.Content = strings.Join(cCtx.Args().Slice(), "\n")
	}
	if strings.TrimSpace(ev.Content) == "" {
		return errors.New("content is empty")
	}

	if sensitive != "" {
		ev.Tags = ev.Tags.AppendUnique(tag.T{"content-warning", sensitive})
	}

	if u == "me" {
		u = ev.PubKey
	}
	var pub string
	if pp := sdk.InputToProfile(context.TODO(), u); pp != nil {
		pub = pp.PublicKey
	} else {
		return fmt.Errorf("failed to parse pubkey from '%s'", u)
	}

	ev.Tags = ev.Tags.AppendUnique(tag.T{"p", pub})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.EncryptedDirectMessage

	ss, err := nip4.ComputeSharedSecret(ev.PubKey, sk)
	if err != nil {
		return err
	}
	ev.Content, err = nip4.Encrypt(ev.Content, ss)
	if err != nil {
		return err
	}
	if err = ev.Sign(sk); err != nil {
		return err
	}

	var success atomic.Int64
	cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		status, err := relay.Publish(ctx, ev)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, relay.URL, status, err)
		}
		if err == nil && status != nostr.PublishStatusFailed {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot post")
	}
	return nil
}

func doPost(cCtx *cli.Context) error {
	stdin := cCtx.Bool("stdin")
	if !stdin && cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	sensitive := cCtx.String("sensitive")
	geohash := cCtx.String("geohash")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	ev := &nip1.Event{}
	if pub, err := nip19.GetPublicKey(sk); err == nil {
		if _, err := nip19.EncodePublicKey(pub); err != nil {
			return err
		}
		ev.PubKey = pub
	} else {
		return err
	}

	if stdin {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		ev.Content = string(b)
	} else {
		ev.Content = strings.Join(cCtx.Args().Slice(), "\n")
	}
	if strings.TrimSpace(ev.Content) == "" {
		return errors.New("content is empty")
	}

	ev.Tags = tags.T{}

	for _, links := range extractLinks(ev.Content) {
		ev.Tags = ev.Tags.AppendUnique(tag.T{"r", links.text})
	}

	for _, u := range cCtx.StringSlice("emoji") {
		tok := strings.SplitN(u, "=", 2)
		if len(tok) != 2 {
			return cli.ShowSubcommandHelp(cCtx)
		}
		ev.Tags = ev.Tags.AppendUnique(tag.T{"emoji", tok[0], tok[1]})
	}
	for _, emojis := range extractEmojis(ev.Content) {
		name := strings.Trim(emojis.text, ":")
		if icon, ok := cfg.Emojis[name]; ok {
			ev.Tags = ev.Tags.AppendUnique(tag.T{"emoji", name, icon})
		}
	}

	for i, u := range cCtx.StringSlice("u") {
		ev.Content = fmt.Sprintf("#[%d] ", i) + ev.Content
		if pp := sdk.InputToProfile(context.TODO(), u); pp != nil {
			u = pp.PublicKey
		} else {
			return fmt.Errorf("failed to parse pubkey from '%s'", u)
		}
		ev.Tags = ev.Tags.AppendUnique(tag.T{"p", u})
	}

	if sensitive != "" {
		ev.Tags = ev.Tags.AppendUnique(tag.T{"content-warning", sensitive})
	}

	if geohash != "" {
		ev.Tags = ev.Tags.AppendUnique(tag.T{"g", geohash})
	}

	hashtag := tag.T{"h"}
	for _, m := range regexp.MustCompile(`#[a-zA-Z0-9]+`).FindAllStringSubmatchIndex(ev.Content, -1) {
		hashtag = append(hashtag, ev.Content[m[0]+1:m[1]])
	}
	if len(hashtag) > 1 {
		ev.Tags = ev.Tags.AppendUnique(hashtag)
	}

	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.TextNote
	if err := ev.Sign(sk); err != nil {
		return err
	}

	var success atomic.Int64
	cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		status, err := relay.Publish(ctx, ev)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, relay.URL, status, err)
		}
		if err == nil && status != nostr.PublishStatusFailed {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot post")
	}
	return nil
}

func doReply(cCtx *cli.Context) error {
	stdin := cCtx.Bool("stdin")
	id := cCtx.String("id")
	quote := cCtx.Bool("quote")
	if !stdin && cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	sensitive := cCtx.String("sensitive")
	geohash := cCtx.String("geohash")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	ev := &nip1.Event{}
	if pub, err := nip19.GetPublicKey(sk); err == nil {
		if _, err := nip19.EncodePublicKey(pub); err != nil {
			return err
		}
		ev.PubKey = pub
	} else {
		return err
	}

	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = evp.ID.String()
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}

	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.TextNote
	if stdin {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		ev.Content = string(b)
	} else {
		ev.Content = strings.Join(cCtx.Args().Slice(), "\n")
	}
	if strings.TrimSpace(ev.Content) == "" {
		return errors.New("content is empty")
	}

	ev.Tags = tags.T{}

	for _, entry := range extractLinks(ev.Content) {
		ev.Tags = ev.Tags.AppendUnique(tag.T{"r", entry.text})
	}

	for _, u := range cCtx.StringSlice("emoji") {
		tok := strings.SplitN(u, "=", 2)
		if len(tok) != 2 {
			return cli.ShowSubcommandHelp(cCtx)
		}
		ev.Tags = ev.Tags.AppendUnique(tag.T{"emoji", tok[0], tok[1]})
	}
	for _, entry := range extractEmojis(ev.Content) {
		name := strings.Trim(entry.text, ":")
		if icon, ok := cfg.Emojis[name]; ok {
			ev.Tags = ev.Tags.AppendUnique(tag.T{"emoji", name, icon})
		}
	}

	if sensitive != "" {
		ev.Tags = ev.Tags.AppendUnique(tag.T{"content-warning", sensitive})
	}

	if geohash != "" {
		ev.Tags = ev.Tags.AppendUnique(tag.T{"g", geohash})
	}

	hashtag := tag.T{"h"}
	for _, m := range regexp.MustCompile(`#[a-zA-Z0-9]+`).FindAllStringSubmatchIndex(ev.Content, -1) {
		hashtag = append(hashtag, ev.Content[m[0]+1:m[1]])
	}
	if len(hashtag) > 1 {
		ev.Tags = ev.Tags.AppendUnique(hashtag)
	}

	var success atomic.Int64
	cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		if !quote {
			ev.Tags = ev.Tags.AppendUnique(tag.T{"e", id, relay.URL, "reply"})
		} else {
			ev.Tags = ev.Tags.AppendUnique(tag.T{"e", id, relay.URL, "mention"})
		}
		if err := ev.Sign(sk); err != nil {
			return true
		}
		status, err := relay.Publish(ctx, ev)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, relay.URL, status, err)
		}
		if err == nil && status != nostr.PublishStatusFailed {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot reply")
	}
	return nil
}

func doRepost(cCtx *cli.Context) error {
	id := cCtx.String("id")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	ev := &nip1.Event{}
	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	if pub, err := nip19.GetPublicKey(sk); err == nil {
		if _, err := nip19.EncodePublicKey(pub); err != nil {
			return err
		}
		ev.PubKey = pub
	} else {
		return err
	}

	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = evp.ID.String()
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	ev.Tags = ev.Tags.AppendUnique(tag.T{"e", id})
	filter := &nip1.Filter{
		Kinds: kinds.T{kind.TextNote},
		IDs:   tag.T{id},
	}

	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.Repost
	ev.Content = ""

	var first atomic.Bool
	first.Store(true)

	var success atomic.Int64
	cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		if first.Load() {
			evs, err := relay.QuerySync(ctx, filter)
			if err != nil {
				return true
			}
			for _, tmp := range evs {
				ev.Tags = ev.Tags.AppendUnique(tag.T{"p", string(tmp.ID)})
			}
			first.Store(false)
			if err := ev.Sign(sk); err != nil {
				return true
			}
		}
		status, err := relay.Publish(ctx, ev)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, relay.URL, status, err)
		}
		if err == nil && status != nostr.PublishStatusFailed {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot repost")
	}
	return nil
}

func doUnrepost(cCtx *cli.Context) error {
	id := cCtx.String("id")
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = evp.ID.String()
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	pub, err := nip19.GetPublicKey(sk)
	if err != nil {
		return err
	}
	filter := &nip1.Filter{
		Kinds:   kinds.T{kind.Repost},
		Authors: tag.T{pub},
		Tags:    nip1.TagMap{"e": tag.T{id}},
	}
	var repostID nip1.EventID
	var mu sync.Mutex
	cfg.Do(relayPerms{Read: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		evs, err := relay.QuerySync(ctx, filter)
		if err != nil {
			return true
		}
		mu.Lock()
		if len(evs) > 0 && repostID == "" {
			repostID = evs[0].ID
		}
		mu.Unlock()
		return true
	})

	var ev *nip1.Event
	ev.Tags = ev.Tags.AppendUnique(tag.T{"e", repostID.String()})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.Deletion
	if err := ev.Sign(sk); err != nil {
		return err
	}

	var success atomic.Int64
	cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		status, err := relay.Publish(ctx, ev)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, relay.URL, status, err)
		}
		if err == nil && status != nostr.PublishStatusFailed {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot unrepost")
	}
	return nil
}

func doLike(cCtx *cli.Context) error {
	id := cCtx.String("id")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	ev := &nip1.Event{}
	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	if pub, err := nip19.GetPublicKey(sk); err == nil {
		if _, err := nip19.EncodePublicKey(pub); err != nil {
			return err
		}
		ev.PubKey = pub
	} else {
		return err
	}

	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = evp.ID.String()
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	ev.Tags = ev.Tags.AppendUnique(tag.T{"e", id})
	filter := &nip1.Filter{
		Kinds: kinds.T{kind.TextNote},
		IDs:   []string{id},
	}

	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.Reaction
	ev.Content = cCtx.String("content")
	emoji := cCtx.String("emoji")
	if emoji != "" {
		if ev.Content == "" {
			ev.Content = "like"
		}
		ev.Tags = ev.Tags.AppendUnique(tag.T{"emoji", ev.Content, emoji})
		ev.Content = ":" + ev.Content + ":"
	}
	if ev.Content == "" {
		ev.Content = "+"
	}

	var first atomic.Bool
	first.Store(true)

	var success atomic.Int64
	cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		if first.Load() {
			evs, err := relay.QuerySync(ctx, filter)
			if err != nil {
				return true
			}
			for _, tmp := range evs {
				ev.Tags = ev.Tags.AppendUnique(tag.T{"p", tmp.ID.String()})
			}
			first.Store(false)
			if err := ev.Sign(sk); err != nil {
				return true
			}
			return true
		}
		status, err := relay.Publish(ctx, ev)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, relay.URL, status, err)
		}
		if err == nil && status != nostr.PublishStatusFailed {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot like")
	}
	return nil
}

func doUnlike(cCtx *cli.Context) error {
	id := cCtx.String("id")
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = evp.ID.String()
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	pub, err := nip19.GetPublicKey(sk)
	if err != nil {
		return err
	}
	filter := &nip1.Filter{
		Kinds:   kinds.T{kind.Reaction},
		Authors: []string{pub},
		Tags:    nip1.TagMap{"e": []string{id}},
	}
	var likeID string
	var mu sync.Mutex
	cfg.Do(relayPerms{Read: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		evs, err := relay.QuerySync(ctx, filter)
		if err != nil {
			return true
		}
		mu.Lock()
		if len(evs) > 0 && likeID == "" {
			likeID = evs[0].ID.String()
		}
		mu.Unlock()
		return true
	})

	var ev *nip1.Event
	ev.Tags = ev.Tags.AppendUnique(tag.T{"e", likeID})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.Deletion
	if err := ev.Sign(sk); err != nil {
		return err
	}

	var success atomic.Int64
	cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		status, err := relay.Publish(ctx, ev)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, relay.URL, status, err)
		}
		if err == nil && status != nostr.PublishStatusFailed {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot unlike")
	}
	return nil
}

func doDelete(cCtx *cli.Context) error {
	id := cCtx.String("id")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	ev := &nip1.Event{}
	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	if pub, err := nip19.GetPublicKey(sk); err == nil {
		if _, err := nip19.EncodePublicKey(pub); err != nil {
			return err
		}
		ev.PubKey = pub
	} else {
		return err
	}

	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = evp.ID.String()
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	ev.Tags = ev.Tags.AppendUnique(tag.T{"e", id})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.Deletion
	if err := ev.Sign(sk); err != nil {
		return err
	}

	var success atomic.Int64
	cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		status, err := relay.Publish(ctx, ev)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, relay.URL, status, err)
		}
		if err == nil && status != nostr.PublishStatusFailed {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot delete")
	}
	return nil
}

func doSearch(cCtx *cli.Context) error {
	n := cCtx.Int("n")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	// get followers
	var followsMap profilesMap
	var err error
	if j && !extra {
		followsMap = make(profilesMap)
	} else {
		followsMap, err = cfg.GetFollows(cCtx.String("a"))
		if err != nil {
			return err
		}
	}

	// get timeline
	filter := &nip1.Filter{
		Kinds:  kinds.T{kind.TextNote},
		Search: strings.Join(cCtx.Args().Slice(), " "),
		Limit:  n,
	}

	evs := cfg.Events(filter)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}

func doStream(cCtx *cli.Context) error {
	kk := cCtx.IntSlice("kind")
	k := kinds.FromIntSlice(kk)
	authors := cCtx.StringSlice("author")
	f := cCtx.Bool("follow")
	pattern := cCtx.String("pattern")
	reply := cCtx.String("reply")

	var re *regexp.Regexp
	if pattern != "" {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return err
		}
	}

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	relay := cfg.FindRelay(context.Background(), relayPerms{Read: true})
	if relay == nil {
		return errors.New("cannot connect relays")
	}
	defer relay.Close()

	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	pub, err := nip19.GetPublicKey(sk)
	if err != nil {
		return err
	}

	// get followers
	var follows []string
	if f {
		followsMap, err := cfg.GetFollows(cCtx.String("a"))
		if err != nil {
			return err
		}
		for k := range followsMap {
			follows = append(follows, k)
		}
	} else {
		follows = authors
	}

	since := timestamp.Now()
	filter := &nip1.Filter{
		Kinds:   k,
		Authors: follows,
		Since:   (*timestamp.Tp)(&since),
	}

	sub, err := relay.Subscribe(context.Background(), nip1.Filters{filter})
	if err != nil {
		return err
	}
	for ev := range sub.Events {
		if ev.Kind == kind.TextNote {
			if re != nil && !re.MatchString(ev.Content) {
				continue
			}
			json.NewEncoder(os.Stdout).Encode(ev)
			if reply != "" {
				var evr *nip1.Event
				evr.PubKey = pub
				evr.Content = reply
				evr.Tags = evr.Tags.AppendUnique(tag.T{"e", ev.ID.String(), "", "reply"})
				evr.CreatedAt = timestamp.Now()
				evr.Kind = kind.TextNote
				if err := evr.Sign(sk); err != nil {
					return err
				}
				cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
					relay.Publish(ctx, evr)
					return true
				})
			}
		} else {
			json.NewEncoder(os.Stdout).Encode(ev)
		}
	}
	return nil
}

func doTimeline(cCtx *cli.Context) error {
	n := cCtx.Int("n")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	// get followers
	followsMap, err := cfg.GetFollows(cCtx.String("a"))
	if err != nil {
		return err
	}
	var follows []string
	for k := range followsMap {
		follows = append(follows, k)
	}

	// get timeline
	filter := &nip1.Filter{
		Kinds:   kinds.T{kind.TextNote},
		Authors: follows,
		Limit:   n,
	}

	evs := cfg.Events(filter)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}

func postMsg(cCtx *cli.Context, msg string) error {
	cfg := cCtx.App.Metadata["config"].(*clientConfig)

	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	ev := &nip1.Event{}
	if pub, err := nip19.GetPublicKey(sk); err == nil {
		if _, err := nip19.EncodePublicKey(pub); err != nil {
			return err
		}
		ev.PubKey = pub
	} else {
		return err
	}

	ev.Content = msg
	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.TextNote
	ev.Tags = tags.T{}
	if err := ev.Sign(sk); err != nil {
		return err
	}

	var success atomic.Int64
	cfg.Do(relayPerms{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		status, err := relay.Publish(ctx, ev)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, relay.URL, status, err)
		}
		if err == nil && status != nostr.PublishStatusFailed {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot post")
	}
	return nil
}

func doPowa(cCtx *cli.Context) error {
	return postMsg(cCtx, "ぽわ〜")
}

func doPuru(cCtx *cli.Context) error {
	return postMsg(cCtx, "(((( ˙꒳​˙  ))))ﾌﾟﾙﾌﾟﾙﾌﾟﾙﾌﾟﾙﾌﾟﾙﾌﾟﾙﾌﾟﾙ")
}
