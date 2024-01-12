package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr-sdk"
	"github.com/urfave/cli/v2"
)

func Reply(cCtx *cli.Context) (e error) {
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
