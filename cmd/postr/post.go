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
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/sdk"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/urfave/cli/v2"
)

func Post(cCtx *cli.Context) (err error) {
	stdin := cCtx.Bool("stdin")
	if !stdin && cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	sensitive, geohash := cCtx.String("sensitive"), cCtx.String("geohash")
	cfg := cCtx.App.Metadata["config"].(*C)
	var pub, sk string
	if pub, sk, err = getPubFromSec(cfg.SecretKey); chk.D(err) {
		return
	}
	ev := &event.T{}
	ev.PubKey = pub
	if stdin {
		var b []byte
		if b, err = io.ReadAll(os.Stdin); chk.D(err) {
			return
		}
		ev.Content = string(b)
	} else {
		ev.Content = strings.Join(cCtx.Args().Slice(), "\n")
	}
	if strings.TrimSpace(ev.Content) == "" {
		return errors.New("content is empty")
	}
	ev.Tags = tags.T{}
	for _, link := range extractLinks(ev.Content) {
		ev.Tags = ev.Tags.AppendUnique(tag.T{"r", link.text})
	}
	for _, u := range cCtx.StringSlice("emoji") {
		tok := strings.SplitN(u, "=", 2)
		if len(tok) != 2 {
			return cli.ShowSubcommandHelp(cCtx)
		}
		ev.Tags = ev.Tags.AppendUnique(tag.T{"emoji", tok[0], tok[1]})
	}
	for _, em := range extractEmojis(ev.Content) {
		emoji := strings.Trim(em.text, ":")
		if icon, ok := cfg.Emojis[emoji]; ok {
			ev.Tags = ev.Tags.AppendUnique(tag.T{"emoji", emoji, icon})
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
	for _, m := range regexp.MustCompile(`#[a-zA-Z0-9]+`).
		FindAllStringSubmatchIndex(ev.Content, -1) {
		hashtag = append(hashtag, ev.Content[m[0]+1:m[1]])
	}
	if len(hashtag) > 1 {
		ev.Tags = ev.Tags.AppendUnique(hashtag)
	}
	ev.CreatedAt = timestamp.Now()
	ev.Kind = kind.TextNote
	log.D.F("signing event `%s`", ev.ToObject())
	if err = ev.Sign(sk); chk.D(err) {
		return err
	}
	var success atomic.Int64
	cfg.Do(writePerms, func(c context.T, rl *client.T) bool {
		err := rl.Publish(c, ev)
		if chk.D(err) {
			log.D.Ln(rl.URL(), err)
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
