package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/sdk"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
)

func doProfile(cCtx *cli.Context) (e error) {
	user := cCtx.String("u")
	j := cCtx.Bool("json")

	cfg := cCtx.App.Metadata["config"].(*clientConfig)
	relay := cfg.FindRelay(context.Background(), relayPerms{Read: true})
	if relay == nil {
		return errors.New("cannot connect relays")
	}
	defer relay.Close()
	var pub string
	if user == "" {
		var s any
		if _, s, e = nip19.Decode(cfg.PrivateKey); fails(e) {
			return
		}
		if pub, e = nip19.GetPublicKey(s.(string)); fails(e) {
			return
		}
	} else {
		if pp := sdk.InputToProfile(context.TODO(), user); pp != nil {
			pub = pp.PublicKey
		} else {
			return fmt.Errorf("failed to parse pubkey from '%s'", user)
		}
	}
	// get set-metadata
	filter := &nip1.Filter{
		Kinds:   kinds.T{kind.ProfileMetadata},
		Authors: tag.T{pub},
		Limit:   1,
	}
	evs := cfg.Events(filter)
	if len(evs) == 0 {
		return errors.New("cannot find user")
	}
	if j {
		fmt.Fprintln(os.Stdout, evs[0].Content)
		return nil
	}
	profile := &userProfile{}
	if e = json.Unmarshal([]byte(evs[0].Content), profile); fails(e) {
		return
	}
	var npub string
	if npub, e = nip19.EncodePublicKey(pub); fails(e) {
		return
	}
	fmt.Printf("Pubkey: %v\n", npub)
	fmt.Printf("Name: %v\n", profile.Name)
	fmt.Printf("DisplayName: %v\n", profile.DisplayName)
	fmt.Printf("WebSite: %v\n", profile.Website)
	fmt.Printf("Picture: %v\n", profile.Picture)
	fmt.Printf("NIP-05: %v\n", profile.Nip05)
	fmt.Printf("LUD-16: %v\n", profile.Lud16)
	fmt.Printf("About: %v\n", profile.About)
	return nil
}
