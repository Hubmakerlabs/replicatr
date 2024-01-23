package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip11"
	"github.com/urfave/cli/v2"
)

var getRelayInfo = &cli.Command{
	Name:  "getinfo",
	Usage: "gets the getRelayInfo information document for the given getRelayInfo, as JSON",
	Description: `example:
		nak getRelayInfo nostr.wine`,
	ArgsUsage: "<relay-url>",
	Action: func(c *cli.Context) error {
		url := c.Args().First()
		if url == "" {
			return fmt.Errorf("specify the <relay-url>")
		}

		if !strings.HasPrefix(url, "wss://") && !strings.HasPrefix(url, "ws://") {
			url = "wss://" + url
		}

		info, err := nip11.Fetch(c.Context, url)
		if err != nil {
			return fmt.Errorf("failed to fetch '%s' information document: %w", url, err)
		}

		pretty, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(pretty))
		return nil
	},
}
