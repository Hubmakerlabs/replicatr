package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/countenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/urfave/cli/v2"
)

var count = &cli.Command{
	Name:        "count",
	Usage:       "generates encoded COUNT messages and optionally use them to talk to relays",
	Description: `outputs a NIP-45 request (the flags are mostly the same as 'nak req').`,
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:     "author",
			Aliases:  []string{"a"},
			Usage:    "only accept events from these authors (pubkey as hex)",
			Category: CategoryFilterAttributes,
		},
		&cli.IntSliceFlag{
			Name:     "kind",
			Aliases:  []string{"k"},
			Usage:    "only accept events with these kind numbers",
			Category: CategoryFilterAttributes,
		},
		&cli.StringSliceFlag{
			Name:     "tag",
			Aliases:  []string{"t"},
			Usage:    "takes a tag like -t e=<id>, only accept events with these tags",
			Category: CategoryFilterAttributes,
		},
		&cli.StringSliceFlag{
			Name:     "e",
			Usage:    "shortcut for --tag e=<value>",
			Category: CategoryFilterAttributes,
		},
		&cli.StringSliceFlag{
			Name:     "p",
			Usage:    "shortcut for --tag p=<value>",
			Category: CategoryFilterAttributes,
		},
		&cli.IntFlag{
			Name:     "since",
			Aliases:  []string{"s"},
			Usage:    "only accept events newer than this (unix timestamp)",
			Category: CategoryFilterAttributes,
		},
		&cli.IntFlag{
			Name:     "until",
			Aliases:  []string{"u"},
			Usage:    "only accept events older than this (unix timestamp)",
			Category: CategoryFilterAttributes,
		},
		&cli.IntFlag{
			Name:     "limit",
			Aliases:  []string{"l"},
			Usage:    "only accept up to this number of events",
			Category: CategoryFilterAttributes,
		},
	},
	ArgsUsage: "[relay...]",
	Action: func(c *cli.Context) error {
		f := filter.T{}

		if authors := c.StringSlice("author"); len(authors) > 0 {
			f.Authors = authors
		}
		if ids := c.StringSlice("id"); len(ids) > 0 {
			f.IDs = ids
		}
		if k := c.IntSlice("kind"); len(k) > 0 {
			f.Kinds = kinds.FromIntSlice(k)
		}
		tags := make([][]string, 0, 5)
		for _, tagFlag := range c.StringSlice("tag") {
			spl := strings.SplitN(tagFlag, "=", 2)
			if len(spl) == 2 && len(spl[0]) == 1 {
				tags = append(tags, spl)
			} else {
				return fmt.Errorf("invalid --tag '%s'", tagFlag)
			}
		}
		for _, etag := range c.StringSlice("e") {
			tags = append(tags, []string{"e", etag})
		}
		for _, ptag := range c.StringSlice("p") {
			tags = append(tags, []string{"p", ptag})
		}
		if len(tags) > 0 {
			f.Tags = make(filter.TagMap)
			for _, tag := range tags {
				if _, ok := f.Tags[tag[0]]; !ok {
					f.Tags[tag[0]] = make([]string, 0, 3)
				}
				f.Tags[tag[0]] = append(f.Tags[tag[0]], tag[1])
			}
		}

		if since := c.Int("since"); since != 0 {
			ts := timestamp.T(since)
			f.Since = ts.Ptr()
		}
		if until := c.Int("until"); until != 0 {
			ts := timestamp.T(until)
			f.Until = ts.Ptr()
		}
		if limit := c.Int("limit"); limit != 0 {
			f.Limit = &limit
		}

		relays := c.Args().Slice()
		successes := 0
		failures := make([]error, 0, len(relays))
		if len(relays) > 0 {
			for _, relayUrl := range relays {
				r, err := relay.Connect(c.Context, relayUrl)
				if err != nil {
					failures = append(failures, err)
					continue
				}
				count, err := r.Count(c.Context, filters.T{&f})
				if err != nil {
					failures = append(failures, err)
					continue
				}
				fmt.Printf("%s: %d", r.URL(), count)
				successes++
			}
			if successes == 0 {
				return errors.Join(failures...)
			}
		} else {
			// no relays given, will just print the filter
			var result string
			result = (&countenvelope.Request{
				ID:      "",
				Filters: filters.T{&f},
			}).ToArray().String()
			// j, _ := json.Marshal({"COUNT", "nak", f})
			// result = string(j)
			fmt.Println(result)
		}

		return nil
	},
}
