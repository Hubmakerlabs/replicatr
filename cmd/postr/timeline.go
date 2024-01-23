package main

import (
	"sync/atomic"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/urfave/cli/v2"
)

func (cfg *C) publish(ev *event.T, s *atomic.Int64) RelayIter {
	return func(c context.T, rl *relay.T) bool {
		e := rl.Publish(c, ev)
		if log.Fail(e) {
			log.D.Ln(rl.URL(), e)
		} else {
			s.Add(1)
		}
		return true
	}
}

func Timeline(cCtx *cli.Context) (e error) {
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
	log.D.Ln("follows", cfg.Follows)
	// get timeline
	f := filter.T{
		Kinds:   kinds.T{kind.TextNote},
		Authors: follows,
		Limit:   n,
	}
	evs := cfg.Events(f)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}
