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
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr-sdk"
	"github.com/urfave/cli/v2"
)

func Post(cCtx *cli.Context) (e error) {
	stdin := cCtx.Bool("stdin")
	if !stdin && cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	sensitive, geohash := cCtx.String("sensitive"), cCtx.String("geohash")
	cfg := cCtx.App.Metadata["config"].(*C)
	var pub, sk string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
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
	cfg.Do(wp, func(c context.T, rl *relay.Relay) bool {
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
