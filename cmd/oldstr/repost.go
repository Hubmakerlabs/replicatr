package main

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr-sdk"
	"github.com/urfave/cli/v2"
)

func Repost(cCtx *cli.Context) (e error) {
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
	cfg.Do(wp, func(c context.T, rl *relay.Relay) bool {
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
