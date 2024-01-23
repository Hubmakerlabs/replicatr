package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nson"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"
)

const CATEGORY_EVENT_FIELDS = "EVENT FIELDS"

var generateEvent = &cli.Command{
	Name:  "event",
	Usage: "generates an encoded event and either prints it or sends it to a set of relays",
	Description: `outputs an event built with the flags. if one or more relays are given as arguments, an attempt is also made to publish the event to these relays.

example:
		nak event -c hello wss://nos.lol
		nak event -k 3 -p 3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d

if an event -- or a partial event -- is given on stdin, the flags can be used to optionally modify it. if it is modified it is rehashed and resigned, otherwise it is just returned as given, but that can be used to just publish to relays.

example:
		echo '{"id":"a889df6a387419ff204305f4c2d296ee328c3cd4f8b62f205648a541b4554dfb","pubkey":"c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5","created_at":1698623783,"kind":1,"tags":[],"content":"hello from the nostr army knife","sig":"84876e1ee3e726da84e5d195eb79358b2b3eaa4d9bd38456fde3e8a2af3f1cd4cda23f23fda454869975b3688797d4c66e12f4c51c1b43c6d2997c5e61865661"}' | nak event wss://offchain.pub
		echo '{"tags": [["t", "spam"]]}' | nak event -c 'this is spam'`,
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
			Name:  "envelope",
			Usage: "print the event enveloped in a [\"EVENT\", ...] message ready to be sent to a getRelayInfo",
		},
		&cli.BoolFlag{
			Name:  "auth",
			Usage: "always perform NIP-42 \"AUTH\" when facing an \"auth-required: \" rejection and try again",
		},
		&cli.BoolFlag{
			Name:  "nson",
			Usage: "encode the event using NSON",
		},
		&cli.IntFlag{
			Name:        "kind",
			Aliases:     []string{"k"},
			Usage:       "event kind",
			DefaultText: "1",
			Value:       0,
			Category:    CATEGORY_EVENT_FIELDS,
		},
		&cli.StringFlag{
			Name:        "content",
			Aliases:     []string{"c"},
			Usage:       "event content",
			DefaultText: "hello from the nostr army knife",
			Value:       "",
			Category:    CATEGORY_EVENT_FIELDS,
		},
		&cli.StringSliceFlag{
			Name:     "tag",
			Aliases:  []string{"t"},
			Usage:    "sets a tag field on the event, takes a value like -t e=<id>",
			Category: CATEGORY_EVENT_FIELDS,
		},
		&cli.StringSliceFlag{
			Name:     "e",
			Usage:    "shortcut for --tag e=<value>",
			Category: CATEGORY_EVENT_FIELDS,
		},
		&cli.StringSliceFlag{
			Name:     "p",
			Usage:    "shortcut for --tag p=<value>",
			Category: CATEGORY_EVENT_FIELDS,
		},
		&cli.StringSliceFlag{
			Name:     "d",
			Usage:    "shortcut for --tag d=<value>",
			Category: CATEGORY_EVENT_FIELDS,
		},
		&cli.StringFlag{
			Name:        "created-at",
			Aliases:     []string{"time", "ts"},
			Usage:       "unix timestamp value for the created_at field",
			DefaultText: "now",
			Value:       "",
			Category:    CATEGORY_EVENT_FIELDS,
		},
	},
	ArgsUsage: "[getRelayInfo...]",
	Action: func(c *cli.Context) error {
		// try to connect to the relays here
		var relays []*relay.T
		if relayUrls := c.Args().Slice(); len(relayUrls) > 0 {
			_, relays = connectToAllRelays(c.Context, relayUrls)
			if len(relays) == 0 {
				log.E.Ln("failed to connect to any of the given relays.")
				os.Exit(3)
			}
		}
		defer func() {
			for _, relay := range relays {
				log.Fail(relay.Close())
			}
		}()
		// gather the secret key
		sec, err := gatherSecretKeyFromArguments(c)
		if err != nil {
			return err
		}
		doAuth := c.Bool("auth")
		// then process input and generate events
		for stdinEvent := range getStdinLinesOrBlank() {
			evt := &event.T{
				Tags: make(tags.T, 0, 3),
			}
			kindWasSupplied := false
			mustRehashAndResign := false

			if stdinEvent != "" {

				if err := json.Unmarshal([]byte(stdinEvent), evt); log.Fail(err) {
					lineProcessingError(c, "invalid event received from stdin: %s", err)
					continue
				}
				kindWasSupplied = strings.Contains(stdinEvent, `"kind"`)
			}

			if k := kind.T(c.Int("kind")); slices.Contains(c.FlagNames(), "kind") {
				evt.Kind = k
				mustRehashAndResign = true
			} else if !kindWasSupplied {
				evt.Kind = 1
				mustRehashAndResign = true
			}

			if content := c.String("content"); content != "" {
				evt.Content = content
				mustRehashAndResign = true
			} else if evt.Content == "" && evt.Kind == 1 {
				evt.Content = "hello from the nostr army knife"
				mustRehashAndResign = true
			}

			tags := make(tags.T, 0, 5)
			for _, tagFlag := range c.StringSlice("tag") {
				// tags are in the format key=value
				tagName, tagValue, found := strings.Cut(tagFlag, "=")
				tag := []string{tagName}
				if found {
					// tags may also contain extra elements separated with a ";"
					tagValues := strings.Split(tagValue, ";")
					tag = append(tag, tagValues...)
					// ~
					tags = tags.AppendUnique(tag)
				}
			}

			for _, etag := range c.StringSlice("e") {
				tags = tags.AppendUnique([]string{"e", etag})
				mustRehashAndResign = true
			}
			for _, ptag := range c.StringSlice("p") {
				tags = tags.AppendUnique([]string{"p", ptag})
				mustRehashAndResign = true
			}
			for _, dtag := range c.StringSlice("d") {
				tags = tags.AppendUnique([]string{"d", dtag})
				mustRehashAndResign = true
			}
			if len(tags) > 0 {
				for _, tag := range tags {
					evt.Tags = append(evt.Tags, tag)
				}
				mustRehashAndResign = true
			}

			if createdAt := c.String("created-at"); createdAt != "" {
				ts := time.Now()
				if createdAt != "now" {
					if v, err := strconv.ParseInt(createdAt, 10, 64); err != nil {
						return fmt.Errorf("failed to parse timestamp '%s': %w", createdAt, err)
					} else {
						ts = time.Unix(v, 0)
					}
				}
				evt.CreatedAt = timestamp.T(ts.Unix())
				mustRehashAndResign = true
			} else if evt.CreatedAt == 0 {
				evt.CreatedAt = timestamp.Now()
				mustRehashAndResign = true
			}

			if evt.Sig == "" || mustRehashAndResign {
				if err := evt.Sign(sec); err != nil {
					return fmt.Errorf("error signing with provided key: %w", err)
				}
			}

			// print event as json
			var result string
			if c.Bool("envelope") {
				result = (&eventenvelope.T{Event: evt}).ToArray().String()
				// j, _ := json.Marshal(eventenvelope.T{Event: evt})
				// result = string(j)
			} else if c.Bool("nson") {
				result, _ = nson.Marshal(evt)
			} else {
				result = evt.ToObject().String()
				// j, _ := json.Marshal(&evt)
				// result = string(j)
			}
			fmt.Println(result)

			// publish to relays
			if len(relays) > 0 {
				log.Fail(os.Stdout.Sync())
				for _, relay := range relays {
				publish:
					log.E.F("publishing to %s... ", relay.URL)
					ctx, cancel := context.WithTimeout(c.Context, 10*time.Second)
					defer cancel()

					err := relay.Publish(ctx, evt)
					if err == nil {
						// published fine
						log.I.Ln("success.")
						continue // continue to next getRelayInfo
					}

					// error publishing
					if strings.HasPrefix(err.Error(), "msg: auth-required:") && sec != "" && doAuth {
						// if the getRelayInfo is requesting auth and we can auth, let's do it
						pk, _ := keys.GetPublicKey(sec)
						log.I.F("performing auth as %s... ", pk)
						if err := relay.Auth(c.Context, func(evt *event.T) error { return evt.Sign(sec) }); err == nil {
							// try to publish again, but this time don't try to auth again
							doAuth = false
							goto publish
						} else {
							log.I.Ln("auth error: %s. ", err)
						}
					}
					log.I.Ln("failed: %s", err)
				}
			}
		}

		exitIfLineProcessingError(c)
		return nil
	},
}
