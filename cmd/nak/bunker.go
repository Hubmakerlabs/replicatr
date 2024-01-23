package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip46"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pool"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"
)

var bunker = &cli.Command{
	Name:        "bunker",
	Usage:       "starts a NIP-46 signer daemon with the given --sec key",
	ArgsUsage:   "[relay...]",
	Description: ``,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "sec",
			Usage:       "secret key to sign the event, as hex or nsec",
			DefaultText: "the key '1'",
			Value:       "0000000000000000000000000000000000000000000000000000000000000001",
		},
		&cli.BoolFlag{
			Name:  "prompt-sec",
			Usage: "prompt the user to paste a hex or nsec with which to sign the event",
		},
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "always respond to any NIP-46 requests from anyone",
		},
	},
	Action: func(c *cli.Context) error {
		// try to connect to the relays here
		qs := url.Values{}
		relayURLs := make([]string, 0, c.Args().Len())
		if relayUrls := c.Args().Slice(); len(relayUrls) > 0 {
			_, relays := connectToAllRelays(c.Context, relayUrls)
			if len(relays) == 0 {
				log.E.F("failed to connect to any of the given relays.")
				os.Exit(3)
			}
			for _, r := range relays {
				relayURLs = append(relayURLs, r.URL())
				qs.Add("relay", r.URL())
			}
		}
		if len(relayURLs) == 0 {
			return fmt.Errorf("not connected to any relays: please specify at least one")
		}

		// gather the secret key
		sec, err := gatherSecretKeyFromArguments(c)
		if err != nil {
			return err
		}
		pubkey, err := keys.GetPublicKey(sec)
		if err != nil {
			return err
		}
		npub, _ := bech32encoding.EncodePublicKey(pubkey)
		log.I.F("listening at %s%v%s:\n  %spubkey:%s %s\n  %snpub:%s %s\n  %sconnection code:%s %s\n  %sbunker:%s %s\n",
			BOLD_ON, relayURLs, BOLD_OFF,
			BOLD_ON, BOLD_OFF, pubkey,
			BOLD_ON, BOLD_OFF, npub,
			BOLD_ON, BOLD_OFF, fmt.Sprintf("%s#secret?%s", npub, qs.Encode()),
			BOLD_ON, BOLD_OFF, fmt.Sprintf("bunker://%s?%s", pubkey, qs.Encode()),
		)

		alwaysYes := c.Bool("yes")

		// subscribe to relays
		pool := pool.NewSimplePool(c.Context)
		events := pool.SubMany(c.Context, relayURLs, filters.T{
			{
				Kinds: kinds.T{kind.NostrConnect},
				Tags:  filter.TagMap{"p": []string{pubkey}},
			},
		}, true)

		signer := nip46.NewStaticKeySigner(sec)
		for ie := range events {
			req, resp, eventResponse, harmless, err := signer.HandleRequest(ie.Event)
			if err != nil {
				log.E.F("< failed to handle request from %s: %s",
					ie.Event.PubKey, err.Error())
				continue
			}

			jreq, _ := json.MarshalIndent(req, "  ", "  ")
			log.I.F("- got request from '%s': %s", ie.Event.PubKey, string(jreq))
			jresp, _ := json.MarshalIndent(resp, "  ", "  ")
			log.I.F("~ responding with %s", string(jresp))

			if alwaysYes || harmless || askProceed(ie.Event.PubKey) {
				if err := ie.Relay.Publish(c.Context, eventResponse); err == nil {
					log.I.F("* sent response!")
				} else {
					log.I.F("* failed to send response: %s", err)
				}
			}
		}

		return nil
	},
}

var allowedSources = make([]string, 0, 2)

func askProceed(source string) bool {
	if slices.Contains(allowedSources, source) {
		return true
	}

	prompt := promptui.Select{
		Label: "proceed?",
		Items: []string{
			"no",
			"yes",
			"always from this source",
		},
	}
	n, _, _ := prompt.Run()
	switch n {
	case 0:
		return false
	case 1:
		return true
	case 2:
		allowedSources = append(allowedSources, source)
		return true
	}

	return false
}
