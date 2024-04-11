package main

import (
	"sync/atomic"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/urfave/cli/v2"
)

func (cfg *C) publish(ev *event.T, s *atomic.Int64) RelayIter {
	return func(c context.T, rl *client.T) bool {
		err := rl.Publish(c, ev)
		if chk.D(err) {
			log.D.Ln(rl.URL(), err)
		} else {
			s.Add(1)
		}
		return true
	}
}

func Timeline(cCtx *cli.Context) (err error) {
	n := cCtx.Int("n")
	log.D.Ln("timeline request count", n)
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")
	cfg := cCtx.App.Metadata["config"].(*C)
	// get followers
	var followsMap Follows
	if followsMap, err = cfg.GetFollows(cCtx.String("a"), cCtx.Bool("update")); chk.D(err) {
		return
	}
	var follows []string
	for k := range followsMap {
		follows = append(follows, k)
	}
	log.D.Ln("follows", cfg.Follows)
	// get timeline
	f := filter.T{
		Kinds:   kinds.T{kind.TextNote},
		Authors: follows,
		Limit:   &n,
	}
	evs := cfg.Events(f)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}
