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

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr-sdk"
	"github.com/urfave/cli/v2"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip04"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
	"github.com/fatih/color"
)

func doDMList(cCtx *cli.Context) (e error) {
	j := cCtx.Bool("json")
	cfg := cCtx.App.Metadata["config"].(*C)
	// get followers
	var followsMap Follows
	followsMap, e = cfg.GetFollows(cCtx.String("a"))
	if log.Fail(e) {
		return e
	}
	var pub string
	if pub, _, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	// get timeline
	f := filter.T{
		Kinds:   []int{event.KindEncryptedDirectMessage},
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

func doDMTimeline(cCtx *cli.Context) (e error) {
	u := cCtx.String("u")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")
	cfg := cCtx.App.Metadata["config"].(*C)
	var npub string
	if npub, _, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
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
	if followsMap, e = cfg.GetFollows(cCtx.String("a")); log.Fail(e) {
		return e
	}
	// get timeline
	f := filter.T{
		Kinds:   []int{event.KindEncryptedDirectMessage},
		Authors: []string{npub, pub},
		Tags:    filter.TagMap{"p": []string{npub, pub}},
		Limit:   9999,
	}
	evs := cfg.Events(f)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}

func doDMPost(cCtx *cli.Context) (e error) {
	u := cCtx.String("u")
	stdin := cCtx.Bool("stdin")
	if !stdin && cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	sensitive := cCtx.String("sensitive")
	cfg := cCtx.App.Metadata["config"].(*C)
	var pubHex, secHex string
	if pubHex, secHex, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	if _, e = nip19.EncodePublicKey(pubHex); log.Fail(e) {
		return e
	}
	ev := &event.T{PubKey: pubHex}
	if stdin {
		var b []byte
		b, e = io.ReadAll(os.Stdin)
		if log.Fail(e) {
			return e
		}
		ev.Content = string(b)
	} else {
		ev.Content = strings.Join(cCtx.Args().Slice(), "\n")
	}
	if strings.TrimSpace(ev.Content) == "" {
		return errors.New("content is empty")
	}
	if sensitive != "" {
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"content-warning", sensitive})
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
	ev.Tags = ev.Tags.AppendUnique(tags.Tag{"p", pub})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = event.KindEncryptedDirectMessage
	var secret []byte
	if secret, e = nip04.ComputeSharedSecret(ev.PubKey, secHex); log.Fail(e) {
		return
	}
	if ev.Content, e = nip04.Encrypt(ev.Content, secret); log.Fail(e) {
		return
	}
	if e = ev.Sign(secHex); log.Fail(e) {
		return
	}
	var success atomic.Int64
	cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
		if e := rl.Publish(c, ev); !log.Fail(e) {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot post")
	}
	return nil
}

func (cfg *C) publish(ev *event.T, success *atomic.Int64) RelayIterator {
	return func(c context.T, rl *relays.Relay) bool {
		e := rl.Publish(c, ev)
		if log.Fail(e) {
			log.D.Ln(rl.URL, e)
		} else {
			success.Add(1)
		}
		return true
	}
}

func doPost(cCtx *cli.Context) (e error) {
	stdin := cCtx.Bool("stdin")
	if !stdin && cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	sensitive, geohash := cCtx.String("sensitive"), cCtx.String("geohash")
	cfg := cCtx.App.Metadata["config"].(*C)
	var pub, sk string
	if pub, sk, e = getPubFromSec(sk); log.Fail(e) {
		return
	}
	ev := &event.T{}
	ev.PubKey = pub
	if stdin {
		var b []byte
		if b, e = io.ReadAll(os.Stdin); log.Fail(e) {
			return
		}
		ev.Content = string(b)
	} else {
		ev.Content = strings.Join(cCtx.Args().Slice(), "\n")
	}
	if strings.TrimSpace(ev.Content) == "" {
		return errors.New("content is empty")
	}
	ev.Tags = tags.Tags{}
	for _, link := range extractLinks(ev.Content) {
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"r", link.text})
	}
	for _, u := range cCtx.StringSlice("emoji") {
		tok := strings.SplitN(u, "=", 2)
		if len(tok) != 2 {
			return cli.ShowSubcommandHelp(cCtx)
		}
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"emoji", tok[0], tok[1]})
	}
	for _, em := range extractEmojis(ev.Content) {
		emoji := strings.Trim(em.text, ":")
		if icon, ok := cfg.Emojis[emoji]; ok {
			ev.Tags = ev.Tags.AppendUnique(tags.Tag{"emoji", emoji, icon})
		}
	}
	for i, u := range cCtx.StringSlice("u") {
		ev.Content = fmt.Sprintf("#[%d] ", i) + ev.Content
		if pp := sdk.InputToProfile(context.TODO(), u); pp != nil {
			u = pp.PublicKey
		} else {
			return fmt.Errorf("failed to parse pubkey from '%s'", u)
		}
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"p", u})
	}
	if sensitive != "" {
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"content-warning", sensitive})
	}
	if geohash != "" {
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"g", geohash})
	}
	hashtag := tags.Tag{"h"}
	for _, m := range regexp.MustCompile(`#[a-zA-Z0-9]+`).FindAllStringSubmatchIndex(ev.Content, -1) {
		hashtag = append(hashtag, ev.Content[m[0]+1:m[1]])
	}
	if len(hashtag) > 1 {
		ev.Tags = ev.Tags.AppendUnique(hashtag)
	}
	ev.CreatedAt = timestamp.Now()
	ev.Kind = event.KindTextNote
	if e = ev.Sign(sk); log.Fail(e) {
		return e
	}
	var success atomic.Int64
	cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
		e := rl.Publish(c, ev)
		if log.Fail(e) {
			log.D.Ln(rl.URL, e)
		} else {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot post")
	}
	return nil
}

func doReply(cCtx *cli.Context) (e error) {
	stdin, id, quote := cCtx.Bool("stdin"), cCtx.String("id"),
		cCtx.Bool("quote")
	if !stdin && cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	sensitive, geohash := cCtx.String("sensitive"), cCtx.String("geohash")
	cfg := cCtx.App.Metadata["config"].(*C)
	var sk, pub string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	ev := &event.T{}
	ev.PubKey = pub
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	ev.CreatedAt = timestamp.Now()
	ev.Kind = event.KindTextNote
	if stdin {
		var b []byte
		if b, e = io.ReadAll(os.Stdin); log.Fail(e) {
			return
		}
		ev.Content = string(b)
	} else {
		ev.Content = strings.Join(cCtx.Args().Slice(), "\n")
	}
	if strings.TrimSpace(ev.Content) == "" {
		return errors.New("content is empty")
	}
	ev.Tags = tags.Tags{}
	for _, link := range extractLinks(ev.Content) {
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"r", link.text})
	}
	for _, u := range cCtx.StringSlice("emoji") {
		tok := strings.SplitN(u, "=", 2)
		if len(tok) != 2 {
			return cli.ShowSubcommandHelp(cCtx)
		}
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"emoji", tok[0], tok[1]})
	}
	for _, em := range extractEmojis(ev.Content) {
		emoji := strings.Trim(em.text, ":")
		if icon, ok := cfg.Emojis[emoji]; ok {
			ev.Tags = ev.Tags.AppendUnique(tags.Tag{"emoji", emoji, icon})
		}
	}
	if sensitive != "" {
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"content-warning", sensitive})
	}
	if geohash != "" {
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"g", geohash})
	}
	hashtag := tags.Tag{"h"}
	for _, m := range regexp.MustCompile(`#[a-zA-Z0-9]+`).FindAllStringSubmatchIndex(ev.Content, -1) {
		hashtag = append(hashtag, ev.Content[m[0]+1:m[1]])
	}
	if len(hashtag) > 1 {
		ev.Tags = ev.Tags.AppendUnique(hashtag)
	}
	var success atomic.Int64
	cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
		if !quote {
			ev.Tags = ev.Tags.AppendUnique(tags.Tag{"e", id, rl.URL, "reply"})
		} else {
			ev.Tags = ev.Tags.AppendUnique(tags.Tag{"e", id, rl.URL, "mention"})
		}
		if e := ev.Sign(sk); log.Fail(e) {
			return true
		}
		if e = rl.Publish(c, ev); log.Fail(e) {
			log.D.Ln(rl.URL, e)
		} else {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot reply")
	}
	return nil
}

func doRepost(cCtx *cli.Context) (e error) {
	id := cCtx.String("id")
	cfg := cCtx.App.Metadata["config"].(*C)
	ev := &event.T{}
	var sk, pub string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	ev.PubKey = pub
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	ev.Tags = ev.Tags.AppendUnique(tags.Tag{"e", id})
	f := filter.T{
		Kinds: []int{event.KindTextNote},
		IDs:   []string{id},
	}
	ev.CreatedAt = timestamp.Now()
	ev.Kind = event.KindRepost
	ev.Content = ""
	var first atomic.Bool
	first.Store(true)
	var success atomic.Int64
	cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
		if first.Load() {
			evs, e := rl.QuerySync(c, f)
			if log.Fail(e) {
				return true
			}
			for _, tmp := range evs {
				ev.Tags = ev.Tags.AppendUnique(tags.Tag{"p", tmp.ID})
			}
			first.Store(false)
			if e = ev.Sign(sk); log.Fail(e) {
				return true
			}
		}
		if e = rl.Publish(c, ev); log.Fail(e) {
			log.D.Ln(rl.URL, e)
		} else {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot repost")
	}
	return nil
}

func doUnrepost(cCtx *cli.Context) (e error) {
	id := cCtx.String("id")
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	cfg := cCtx.App.Metadata["config"].(*C)
	var sk, pub string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	f := filter.T{
		Kinds:   []int{event.KindRepost},
		Authors: []string{pub},
		Tags:    filter.TagMap{"e": []string{id}},
	}
	var repostID string
	var mu sync.Mutex
	cfg.Do(rp, func(c context.T, rl *relays.Relay) bool {
		evs, e := rl.QuerySync(c, f)
		if log.Fail(e) {
			return true
		}
		mu.Lock()
		if len(evs) > 0 && repostID == "" {
			repostID = evs[0].ID
		}
		mu.Unlock()
		return true
	})
	ev := &event.T{}
	ev.Tags = ev.Tags.AppendUnique(tags.Tag{"e", repostID})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = event.KindDeletion
	if e = ev.Sign(sk); log.Fail(e) {
		return e
	}
	var success atomic.Int64
	cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
		e := rl.Publish(c, ev)
		if log.Fail(e) {
			log.D.Ln(rl.URL, e)
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

func doLike(cCtx *cli.Context) (e error) {
	id := cCtx.String("id")
	cfg := cCtx.App.Metadata["config"].(*C)
	ev := &event.T{}
	var sk, pub string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	ev.PubKey = pub
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	ev.Tags = ev.Tags.AppendUnique(tags.Tag{"e", id})
	f := filter.T{
		Kinds: []int{event.KindTextNote},
		IDs:   []string{id},
	}
	ev.CreatedAt = timestamp.Now()
	ev.Kind = event.KindReaction
	ev.Content = cCtx.String("content")
	emoji := cCtx.String("emoji")
	if emoji != "" {
		if ev.Content == "" {
			ev.Content = "like"
		}
		ev.Tags = ev.Tags.AppendUnique(tags.Tag{"emoji", ev.Content, emoji})
		ev.Content = ":" + ev.Content + ":"
	}
	if ev.Content == "" {
		ev.Content = "+"
	}
	var first atomic.Bool
	first.Store(true)
	var success atomic.Int64
	cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
		if first.Load() {
			evs, e := rl.QuerySync(c, f)
			if log.Fail(e) {
				return true
			}
			for _, tmp := range evs {
				ev.Tags = ev.Tags.AppendUnique(tags.Tag{"p", tmp.ID})
			}
			first.Store(false)
			if e = ev.Sign(sk); log.Fail(e) {
				return true
			}
			return true
		}
		if e := rl.Publish(c, ev); log.Fail(e) {
			log.D.Ln(rl.URL, e)
		} else {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot like")
	}
	return nil
}

func doUnlike(cCtx *cli.Context) (e error) {
	id := cCtx.String("id")
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	cfg := cCtx.App.Metadata["config"].(*C)
	var sk, pub string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	f := filter.T{
		Kinds:   []int{event.KindReaction},
		Authors: []string{pub},
		Tags:    filter.TagMap{"e": []string{id}},
	}
	var likeID string
	var mu sync.Mutex
	cfg.Do(rp, func(c context.T, rl *relays.Relay) bool {
		evs, e := rl.QuerySync(c, f)
		if log.Fail(e) {
			return true
		}
		mu.Lock()
		if len(evs) > 0 && likeID == "" {
			likeID = evs[0].ID
		}
		mu.Unlock()
		return true
	})
	ev := &event.T{}
	ev.Tags = ev.Tags.AppendUnique(tags.Tag{"e", likeID})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = event.KindDeletion
	if e := ev.Sign(sk); log.Fail(e) {
		return e
	}
	var success atomic.Int64
	cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
		e := rl.Publish(c, ev)
		if !log.Fail(e) {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot unlike")
	}
	return nil
}

func doDelete(cCtx *cli.Context) (e error) {
	id := cCtx.String("id")
	cfg := cCtx.App.Metadata["config"].(*C)
	ev := &event.T{}
	var pub, sk string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	ev.PubKey = pub
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	ev.Tags = ev.Tags.AppendUnique(tags.Tag{"e", id})
	ev.CreatedAt = timestamp.Now()
	ev.Kind = event.KindDeletion
	if e = ev.Sign(sk); log.Fail(e) {
		return e
	}
	var success atomic.Int64
	cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
		e := rl.Publish(c, ev)
		if !log.Fail(e) {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot delete")
	}
	return nil
}

func doSearch(cCtx *cli.Context) (e error) {
	n := cCtx.Int("n")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")
	cfg := cCtx.App.Metadata["config"].(*C)
	// get followers
	var followsMap Follows
	if j && !extra {
		followsMap = make(Follows)
	} else {
		followsMap, e = cfg.GetFollows(cCtx.String("a"))
		if log.Fail(e) {
			return e
		}
	}
	// get timeline
	f := filter.T{
		Kinds:  []int{event.KindTextNote},
		Search: strings.Join(cCtx.Args().Slice(), " "),
		Limit:  n,
	}
	evs := cfg.Events(f)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}

func doStream(cCtx *cli.Context) (e error) {
	kinds := cCtx.IntSlice("kind")
	authors := cCtx.StringSlice("author")
	f := cCtx.Bool("follow")
	pattern := cCtx.String("pattern")
	reply := cCtx.String("reply")
	var re *regexp.Regexp
	if pattern != "" {
		var e error
		re, e = regexp.Compile(pattern)
		if log.Fail(e) {
			return e
		}
	}
	cfg := cCtx.App.Metadata["config"].(*C)
	rl := cfg.FindRelay(context.Bg(), &RelayPerms{Read: true})
	if rl == nil {
		return errors.New("cannot connect relays")
	}
	defer log.Fail(rl.Close())
	var pub, sk string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	// get followers
	var follows []string
	if f {
		followsMap := make(Follows)
		if followsMap, e = cfg.GetFollows(cCtx.String("a")); log.Fail(e) {
			return
		}
		for k := range followsMap {
			follows = append(follows, k)
		}
	} else {
		follows = authors
	}
	since := timestamp.Now()
	ff := filter.T{
		Kinds:   kinds,
		Authors: follows,
		Since:   &since,
	}
	var sub *relays.Subscription
	sub, e = rl.Subscribe(context.Bg(), filters.T{ff})
	if log.Fail(e) {
		return e
	}
	for ev := range sub.Events {
		if ev.Kind == event.KindTextNote {
			if re != nil && !re.MatchString(ev.Content) {
				continue
			}
			log.Fail(json.NewEncoder(os.Stdout).Encode(ev))
			if reply != "" {
				evr := &event.T{}
				evr.PubKey = pub
				evr.Content = reply
				evr.Tags = evr.Tags.AppendUnique(tags.Tag{"e", ev.ID, "", "reply"})
				evr.CreatedAt = timestamp.Now()
				evr.Kind = event.KindTextNote
				if e := evr.Sign(sk); log.Fail(e) {
					return e
				}
				cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
					log.Fail(rl.Publish(c, evr))
					return true
				})
			}
		} else {
			log.Fail(json.NewEncoder(os.Stdout).Encode(ev))
		}
	}
	return nil
}

func doTimeline(cCtx *cli.Context) (e error) {
	n := cCtx.Int("n")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")
	cfg := cCtx.App.Metadata["config"].(*C)
	// get followers
	var followsMap Follows
	if followsMap, e = cfg.GetFollows(cCtx.String("a")); log.Fail(e) {
		return
	}
	var follows []string
	for k := range followsMap {
		follows = append(follows, k)
	}
	// get timeline
	f := filter.T{
		Kinds:   []int{event.KindTextNote},
		Authors: follows,
		Limit:   n,
	}
	evs := cfg.Events(f)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}

func postMsg(cCtx *cli.Context, msg string) (e error) {
	cfg := cCtx.App.Metadata["config"].(*C)
	var pub, sk string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	ev := &event.T{
		PubKey:    pub,
		CreatedAt: timestamp.Now(),
		Kind:      event.KindTextNote,
		Tags:      tags.Tags{},
		Content:   msg,
		Sig:       "",
	}
	if e = ev.Sign(sk); log.Fail(e) {
		return e
	}
	var success atomic.Int64
	cfg.Do(wp, func(c context.T, rl *relays.Relay) bool {
		e := rl.Publish(c, ev)
		if log.Fail(e) {
			log.D.Ln(rl.URL, e)
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
