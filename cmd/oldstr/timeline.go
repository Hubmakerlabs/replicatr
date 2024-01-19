package main

import (
	"sync/atomic"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/urfave/cli/v2"
)

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
	log.D.Ln("follows",follows)
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

