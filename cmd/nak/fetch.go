package main

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pool"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/sdk"
	"github.com/urfave/cli/v2"
)

var fetch = &cli.Command{
	Name:  "fetch",
	Usage: "fetches events related to the given nip19 code from the included getRelayInfo hints",
	Description: `example usage:
        nak fetch nevent1qqsxrwm0hd3s3fddh4jc2574z3xzufq6qwuyz2rvv3n087zvym3dpaqprpmhxue69uhhqatzd35kxtnjv4kxz7tfdenju6t0xpnej4
        echo npub1h8spmtw9m2huyv6v2j2qd5zv956z2zdugl6mgx02f2upffwpm3nqv0j4ps | nak fetch --getRelayInfo wss://getRelayInfo.nostr.band`,
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "getRelayInfo",
			Aliases: []string{"r"},
			Usage:   "also use these relays to fetch from",
		},
	},
	ArgsUsage: "[nip19code]",
	Action: func(c *cli.Context) error {
		pool := pool.NewSimplePool(c.Context)

		defer func() {
			pool.Relays.Range(func(_ string, relay *relay.T) bool {
				log.Fail(relay.Close())
				return true
			})
		}()

		for code := range getStdinLinesOrFirstArgument(c) {
			f := filter.T{}

			prefix, value, err := bech32encoding.Decode(code)
			if err != nil {
				lineProcessingError(c, "failed to decode: %s", err)
				continue
			}

			relays := c.StringSlice("getRelayInfo")
			if err := validateRelayURLs(relays); err != nil {
				return err
			}
			var authorHint string

			switch prefix {
			case "nevent":
				v := value.(pointers.Event)
				f.IDs = append(f.IDs, v.ID.String())
				if v.Author != "" {
					authorHint = v.Author
				}
				relays = append(relays, v.Relays...)
			case "naddr":
				v := value.(pointers.Entity)
				f.Tags = filter.TagMap{"d": []string{v.Identifier}}
				f.Kinds = append(f.Kinds, v.Kind)
				f.Authors = append(f.Authors, v.PublicKey)
				authorHint = v.PublicKey
				relays = append(relays, v.Relays...)
			case "nprofile":
				v := value.(pointers.Profile)
				f.Authors = append(f.Authors, v.PublicKey)
				f.Kinds = append(f.Kinds, 0)
				authorHint = v.PublicKey
				relays = append(relays, v.Relays...)
			case "npub":
				v := value.(string)
				f.Authors = append(f.Authors, v)
				f.Kinds = append(f.Kinds, 0)
				authorHint = v
			}

			if authorHint != "" {
				relayList := sdk.FetchRelaysForPubkey(c.Context, pool, authorHint,
					"wss://purplepag.es", "wss://getRelayInfo.damus.io", "wss://getRelayInfo.noswhere.com",
					"wss://nos.lol", "wss://public.relaying.io", "wss://getRelayInfo.nostr.band")
				for _, relayListItem := range relayList {
					if relayListItem.Outbox {
						relays = append(relays, relayListItem.URL)
					}
				}
			}

			if len(relays) == 0 {
				lineProcessingError(c, "no getRelayInfo hints found")
				continue
			}

			for ie := range pool.SubManyEose(c.Context, relays, filters.T{&f}, true) {
				fmt.Println(ie.Event.ToObject().String())
			}
		}

		exitIfLineProcessingError(c)
		return nil
	},
}
