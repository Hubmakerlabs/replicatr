package main

import (
	"github.com/urfave/cli/v2"
)

func Get(cCtx *cli.Context) (e error) {
	asJSON := cCtx.Bool("json")
	ids := cCtx.Args().Slice()
	cfg := cCtx.App.Metadata["config"].(*C)
	evs := cfg.GetEvents(ids)
	log.D.S(evs)
	if len(evs) > 0 {
		cfg.PrintEvents(evs, nil, asJSON, false)
	}
	return
}
