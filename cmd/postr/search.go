package main

import (
	"strings"

	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/kinds"
	"github.com/urfave/cli/v2"
)

func Search(cCtx *cli.Context) (err error) {
	n := cCtx.Int("n")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")
	cfg := cCtx.App.Metadata["config"].(*C)
	// get followers
	var followsMap Follows
	if j && !extra {
		followsMap = make(Follows)
	} else {
		followsMap, err = cfg.GetFollows(cCtx.String("a"), false)
		if chk.D(err) {
			return err
		}
	}
	// get timeline
	f := filter.T{
		Kinds:  kinds.T{kind.TextNote},
		Search: strings.Join(cCtx.Args().Slice(), " "),
		Limit:  &n,
	}
	evs := cfg.Events(f)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}
