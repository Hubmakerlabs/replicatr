package main

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/sdk"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/urfave/cli/v2"
)

func Like(cCtx *cli.Context) (err error) {
	id := cCtx.String("id")
	cfg := cCtx.App.Metadata["config"].(*C)
	ev := &event.T{}
	var sk, pub string
	if pub, sk, err = getPubFromSec(cfg.SecretKey); chk.D(err) {
		return
	}
	ev.PubKey = pub
	if evp := sdk.InputToEventPointer(id); evp != nil {
		id = string(evp.ID)
	} else {
		return fmt.Errorf("failed to parse event from '%s'", id)
	}
	ev.Tags = ev.Tags.AppendUnique(tag.T{"e", id})
	f := filter.T{
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
	cfg.Do(writePerms, func(c context.T, rl *client.T) bool {
		if first.Load() {
			var evs []*event.T
			evs, err = rl.QuerySync(c, &f)
			if chk.D(err) {
				return true
			}
			for _, tmp := range evs {
				ev.Tags = ev.Tags.AppendUnique(tag.T{"p", tmp.ID.String()})
			}
			first.Store(false)
			if err = ev.Sign(sk); chk.D(err) {
				return true
			}
			return true
		}
		if err = rl.Publish(c, ev); chk.D(err) {
			log.D.Ln(rl.URL(), err)
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
