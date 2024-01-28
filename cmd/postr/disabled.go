package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip4"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/sdk"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscription"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func postMsg(cCtx *cli.Context, msg string) (err error) {
	cfg := cCtx.App.Metadata["config"].(*C)
	var pub, sk string
	if pub, sk, err = getPubFromSec(cfg.SecretKey); log.Fail(err) {
		return
	}
	ev := &event.T{
		PubKey:    pub,
		CreatedAt: timestamp.Now(),
		Kind:      kind.TextNote,
		Tags:      tags.T{},
		Content:   msg,
		Sig:       "",
	}
	if err = ev.Sign(sk); log.Fail(err) {
		return err
	}
	var success atomic.Int64
	cfg.Do(writePerms, func(c context.T, rl *relay.T) bool {
		err := rl.Publish(c, ev)
		if log.Fail(err) {
			log.D.Ln(rl.URL(), err)
		} else {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot post")
	}
	return
}

func doDMList(cCtx *cli.Context) (err error) {
	j := cCtx.Bool("json")
	cfg := cCtx.App.Metadata["config"].(*C)
	// get followers
	var followsMap Follows
	followsMap, err = cfg.GetFollows(cCtx.String("a"), false)
	if log.Fail(err) {
		return err
	}
	var pub string
	if pub, _, err = getPubFromSec(cfg.SecretKey); log.Fail(err) {
		return
	}
	// get timeline
	f := filter.T{
		Kinds:   kinds.T{kind.EncryptedDirectMessage},
		Authors: []string{pub},
	}
	evs := cfg.Events(f)
	type entry struct {
		name   string
		pubkey string
	}
	var users []entry
	m := Checklist{}
	for _, ev := range evs {
		p := ev.Tags.GetFirst([]string{"p"}).Value()
		if _, ok := m[p]; ok {
			continue
		}
		if profile, ok := followsMap[p]; ok {
			m[p] = struct{}{}
			p, _ = bech32encoding.EncodePublicKey(p)
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
			log.Fail(json.NewEncoder(os.Stdout).Encode(user))
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

func doDMTimeline(cCtx *cli.Context) (err error) {
	u := cCtx.String("u")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")
	cfg := cCtx.App.Metadata["config"].(*C)
	var npub string
	if npub, _, err = getPubFromSec(cfg.SecretKey); log.Fail(err) {
		return
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
	var followsMap Follows
	if followsMap, err = cfg.GetFollows(cCtx.String("a"), false); log.Fail(err) {
		return err
	}
	// get timeline
	f := filter.T{
		Kinds:   kinds.T{kind.EncryptedDirectMessage},
		Authors: []string{npub, pub},
		Tags:    filter.TagMap{"p": []string{npub, pub}},
		Limit:   9999,
	}
	evs := cfg.Events(f)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}

func doDMPost(cCtx *cli.Context) (err error) {
	u := cCtx.String("u")
	stdin := cCtx.Bool("stdin")
	if !stdin && cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	sensitive := cCtx.String("sensitive")
	cfg := cCtx.App.Metadata["config"].(*C)
	var pubHex, secHex string
	if pubHex, secHex, err = getPubFromSec(cfg.SecretKey); log.Fail(err) {
		return
	}
	if _, err = bech32encoding.EncodePublicKey(pubHex); log.Fail(err) {
		return err
	}
	ev := &event.T{PubKey: pubHex}
	if stdin {
		var b []byte
		b, err = io.ReadAll(os.Stdin)
		if log.Fail(err) {
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
	var secret []byte
	if secret, err = nip4.ComputeSharedSecret(ev.PubKey, secHex); log.Fail(err) {
		return
	}
	if ev.Content, err = nip4.Encrypt(ev.Content, secret); log.Fail(err) {
		return
	}
	if err = ev.Sign(secHex); log.Fail(err) {
		return
	}
	var success atomic.Int64
	cfg.Do(writePerms, func(c context.T, rl *relay.T) bool {
		if err := rl.Publish(c, ev); !log.Fail(err) {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot post")
	}
	return nil
}

func doUnrepost(cCtx *cli.Context) (err error) {
	id := cCtx.String("id")
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	cfg := cCtx.App.Metadata["config"].(*C)
	var sk, pub string
	if pub, sk, err = getPubFromSec(cfg.SecretKey); log.Fail(err) {
		return
	}
	f := filter.T{
		Kinds:   kinds.T{kind.Repost},
		Authors: []string{pub},
		Tags:    filter.TagMap{"e": []string{id}},
	}
	var repostID string
	var mu sync.Mutex
	cfg.Do(readPerms, func(c context.T, rl *relay.T) bool {
		evs, err := rl.QuerySync(c, &f)
		if log.Fail(err) {
			return true
		}
		mu.Lock()
		if len(evs) > 0 && repostID == "" {
			repostID = evs[0].ID.String()
		}
		mu.Unlock()
		return true
	})
	ev := &event.T{}
	ev.Tags = ev.Tags.AppendUnique(tag.T{"e", repostID})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.Deletion
	if err = ev.Sign(sk); log.Fail(err) {
		return err
	}
	var success atomic.Int64
	cfg.Do(writePerms, func(c context.T, rl *relay.T) bool {
		err := rl.Publish(c, ev)
		if log.Fail(err) {
			log.D.Ln(rl.URL(), err)
		} else {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot unrepost")
	}
	return nil
}

func doUnlike(cCtx *cli.Context) (err error) {
	id := cCtx.String("id")
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	cfg := cCtx.App.Metadata["config"].(*C)
	var sk, pub string
	if pub, sk, err = getPubFromSec(cfg.SecretKey); log.Fail(err) {
		return
	}
	f := filter.T{
		Kinds:   kinds.T{kind.Reaction},
		Authors: []string{pub},
		Tags:    filter.TagMap{"e": []string{id}},
	}
	var likeID string
	var mu sync.Mutex
	cfg.Do(readPerms, func(c context.T, rl *relay.T) bool {
		evs, err := rl.QuerySync(c, &f)
		if log.Fail(err) {
			return true
		}
		mu.Lock()
		if len(evs) > 0 && likeID == "" {
			likeID = evs[0].ID.String()
		}
		mu.Unlock()
		return true
	})
	ev := &event.T{}
	ev.Tags = ev.Tags.AppendUnique(tag.T{"e", likeID})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.Deletion
	if err := ev.Sign(sk); log.Fail(err) {
		return err
	}
	var success atomic.Int64
	cfg.Do(writePerms, func(c context.T, rl *relay.T) bool {
		err := rl.Publish(c, ev)
		if !log.Fail(err) {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot unlike")
	}
	return nil
}

func doDelete(cCtx *cli.Context) (err error) {
	id := cCtx.String("id")
	cfg := cCtx.App.Metadata["config"].(*C)
	ev := &event.T{}
	var pub, sk string
	if pub, sk, err = getPubFromSec(cfg.SecretKey); log.Fail(err) {
		return
	}
	ev.PubKey = pub
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	ev.Tags = ev.Tags.AppendUnique(tag.T{"e", id})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.Deletion
	if err = ev.Sign(sk); log.Fail(err) {
		return err
	}
	var success atomic.Int64
	cfg.Do(writePerms, func(c context.T, rl *relay.T) bool {
		err := rl.Publish(c, ev)
		if !log.Fail(err) {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot delete")
	}
	return nil
}

func doStream(cCtx *cli.Context) (err error) {
	k := cCtx.IntSlice("kind")
	authors := cCtx.StringSlice("author")
	f := cCtx.Bool("follow")
	pattern := cCtx.String("pattern")
	reply := cCtx.String("reply")
	var re *regexp.Regexp
	if pattern != "" {
		var err error
		re, err = regexp.Compile(pattern)
		if log.Fail(err) {
			return err
		}
	}
	cfg := cCtx.App.Metadata["config"].(*C)
	rl := cfg.FindRelay(context.Bg(), &RelayPerms{Read: true})
	if rl == nil {
		return errors.New("cannot connect relays")
	}
	defer log.Fail(rl.Close())
	var pub, sk string
	if pub, sk, err = getPubFromSec(cfg.SecretKey); log.Fail(err) {
		return
	}
	// get followers
	var follows []string
	if f {
		followsMap := make(Follows)
		if followsMap, err = cfg.GetFollows(cCtx.String("a"), false); log.Fail(err) {
			return
		}
		for k := range followsMap {
			follows = append(follows, k)
		}
	} else {
		follows = authors
	}
	since := timestamp.Now().Ptr()
	ff := filter.T{
		Kinds:   kinds.FromIntSlice(k),
		Authors: follows,
		Since:   since,
	}
	var sub *subscription.T
	sub, err = rl.Subscribe(context.Bg(), filters.T{&ff})
	if log.Fail(err) {
		return err
	}
	for ev := range sub.Events {
		if ev.Kind == kind.TextNote {
			if re != nil && !re.MatchString(ev.Content) {
				continue
			}
			log.Fail(json.NewEncoder(os.Stdout).Encode(ev))
			if reply != "" {
				evr := &event.T{}
				evr.PubKey = pub
				evr.Content = reply
				evr.Tags = evr.Tags.AppendUnique(tag.T{"e", ev.ID.String(), "",
					"reply"})
				evr.CreatedAt = timestamp.Now()
				evr.Kind = kind.TextNote
				if err := evr.Sign(sk); log.Fail(err) {
					return err
				}
				cfg.Do(writePerms, func(c context.T, rl *relay.T) bool {
					if err = rl.Publish(c, evr); log.Fail(err) {
					}
					return true
				})
			}
		} else {
			log.Fail(json.NewEncoder(os.Stdout).Encode(ev))
		}
	}
	return nil
}
