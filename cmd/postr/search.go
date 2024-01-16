package main

import (
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/urfave/cli/v2"
)

func Search(cCtx *cli.Context) (e error) {
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
		Kinds:  kinds.T{kind.TextNote},
		Search: strings.Join(cCtx.Args().Slice(), " "),
		Limit:  n,
	}
	evs := cfg.Events(f)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}
