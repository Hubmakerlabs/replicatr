package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/reqenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pool"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/urfave/cli/v2"
)

const CategoryFilterAttributes = "FILTER ATTRIBUTES"

var req = &cli.Command{
	Name:  "req",
	Usage: "generates encoded REQ messages and optionally use them to talk to relays",
	Description: `outputs a NIP-01 Nostr filter. when a relay is not given, will print the filter, otherwise will connect to the given relay and send the filter.

example:
		nak req -k 1 -l 15 wss://nostr.wine wss://nostr-pub.wellorder.net
		nak req -k 0 -a 3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d wss://nos.lol | jq '.content | fromjson | .name'

it can also take a filter from stdin, optionally modify it with flags and send it to specific relays (or just print it).

example:
		echo '{"kinds": [1], "#t": ["test"]}' | nak req -l 5 -k 4549 --tag t=spam wss://nostr-pub.wellorder.net`,
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:     "author",
			Aliases:  []string{"a"},
			Usage:    "only accept events from these authors (pubkey as hex)",
			Category: CategoryFilterAttributes,
		},
		&cli.StringSliceFlag{
			Name:     "id",
			Aliases:  []string{"i"},
			Usage:    "only accept events with these ids (hex)",
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
		&cli.StringSliceFlag{
			Name:     "d",
			Usage:    "shortcut for --tag d=<value>",
			Category: CategoryFilterAttributes,
		},
		&cli.StringFlag{
			Name:     "since",
			Aliases:  []string{"s"},
			Usage:    "only accept events newer than this (unix timestamp)",
			Category: CategoryFilterAttributes,
		},
		&cli.StringFlag{
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
		&cli.StringFlag{
			Name:     "search",
			Usage:    "a NIP-50 search query, use it only with relays that explicitly support it",
			Category: CategoryFilterAttributes,
		},
		&cli.BoolFlag{
			Name:        "stream",
			Usage:       "keep the subscription open, printing all events as they are returned",
			DefaultText: "false, will close on EOSE",
		},
		&cli.BoolFlag{
			Name:  "bare",
			Usage: "when printing the filter, print just the filter, not enveloped in a [\"REQ\", ...] array",
		},
		&cli.BoolFlag{
			Name:  "auth",
			Usage: "always perform NIP-42 \"AUTH\" when facing an \"auth-required: \" rejection and try again",
		},
		&cli.StringFlag{
			Name:        "sec",
			Usage:       "secret key to sign the AUTH challenge, as hex or nsec",
			DefaultText: "the key '1'",
			Value:       "0000000000000000000000000000000000000000000000000000000000000001",
		},
		&cli.BoolFlag{
			Name:  "prompt-sec",
			Usage: "prompt the user to paste a hex or nsec with which to sign the AUTH challenge",
		},
	},
	ArgsUsage: "[relay...]",
	Action: func(c *cli.Context) error {
		var p *pool.Simple
		relayUrls := c.Args().Slice()
		if len(relayUrls) > 0 {
			var relays []*relay.T
			p, relays = connectToAllRelays(c.Context, relayUrls, pool.WithAuthHandler(func(evt *event.T) error {
				if !c.Bool("auth") {
					return fmt.Errorf("auth not authorized")
				}
				sec, err := gatherSecretKeyFromArguments(c)
				if err != nil {
					return err
				}
				pk, _ := keys.GetPublicKey(sec)
				log.I.Ln("performing auth as %s...", pk)
				return evt.Sign(sec)
			}))
			if len(relays) == 0 {
				log.E.Ln("failed to connect to any of the given relays.")
				os.Exit(3)
			}
			relayUrls = make([]string, len(relays))
			for i, relay := range relays {
				relayUrls[i] = relay.URL()
			}

			defer func() {
				for _, relay := range relays {
					relay.Close()
				}
			}()
		}

		for stdinFilter := range getStdinLinesOrBlank() {
			f := &filter.T{}
			if stdinFilter != "" {
				if err := json.Unmarshal([]byte(stdinFilter), f); err != nil {
					lineProcessingError(c, "invalid filter '%s' received from stdin: %s", stdinFilter, err)
					continue
				}
			}

			if authors := c.StringSlice("author"); len(authors) > 0 {
				f.Authors = append(f.Authors, authors...)
			}
			if ids := c.StringSlice("id"); len(ids) > 0 {
				f.IDs = append(f.IDs, ids...)
			}
			if kinds := kinds.FromIntSlice(c.IntSlice("kind")); len(kinds) > 0 {
				f.Kinds = append(f.Kinds, kinds...)
			}
			if search := c.String("search"); search != "" {
				f.Search = search
			}
			tags := make([][]string, 0, 5)
			for _, tagFlag := range c.StringSlice("tag") {
				spl := strings.Split(tagFlag, "=")
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
			for _, dtag := range c.StringSlice("d") {
				tags = append(tags, []string{"d", dtag})
			}

			if len(tags) > 0 && f.Tags == nil {
				f.Tags = make(filter.TagMap)
			}

			for _, tag := range tags {
				if _, ok := f.Tags[tag[0]]; !ok {
					f.Tags[tag[0]] = make([]string, 0, 3)
				}
				f.Tags[tag[0]] = append(f.Tags[tag[0]], tag[1])
			}

			if since := c.String("since"); since != "" {
				if since == "now" {
					ts := timestamp.Now()
					f.Since = ts.Ptr()
				} else if i, err := strconv.Atoi(since); err == nil {
					ts := timestamp.T(i)
					f.Since = ts.Ptr()
				} else {
					return fmt.Errorf("parse error: Invalid numeric literal %q", since)
				}
			}
			if until := c.String("until"); until != "" {
				if until == "now" {
					ts := timestamp.Now()
					f.Until = ts.Ptr()
				} else if i, err := strconv.Atoi(until); err == nil {
					ts := timestamp.T(i)
					f.Until = ts.Ptr()
				} else {
					return fmt.Errorf("parse error: Invalid numeric literal %q", until)
				}
			}
			if limit := c.Int("limit"); limit != 0 {
				f.Limit = limit
			}

			if len(relayUrls) > 0 {
				fn := p.SubManyEose
				if c.Bool("stream") {
					fn = p.SubMany
				}
				for ie := range fn(c.Context, relayUrls, filters.T{f}, true) {
					fmt.Println(ie.Event)
				}
			} else {
				// no relays given, will just print the filter
				var result string
				if c.Bool("bare") {
					result = f.String()
				} else {
					result = (&reqenvelope.T{
						SubscriptionID: "nak",
						Filters:        filters.T{f},
					}).ToArray().String()
				}

				fmt.Println(result)
			}
		}

		exitIfLineProcessingError(c)
		return nil
	},
}
