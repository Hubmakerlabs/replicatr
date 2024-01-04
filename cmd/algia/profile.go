package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr-sdk"
	"github.com/urfave/cli/v2"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
)

func doProfile(cCtx *cli.Context) (e error) {
	user := cCtx.String("u")
	j := cCtx.Bool("json")

	cfg := cCtx.App.Metadata["config"].(*Config)
	relay := cfg.FindRelay(context.Background(), RelayPerms{Read: true})
	if relay == nil {
		return errors.New("cannot connect relays")
	}
	defer relay.Close()

	var pub string
	if user == "" {
		if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
			if pub, err = keys.GetPublicKey(s.(string)); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if pp := sdk.InputToProfile(context.TODO(), user); pp != nil {
			pub = pp.PublicKey
		} else {
			return fmt.Errorf("failed to parse pubkey from '%s'", user)
		}
	}

	// get set-metadata
	f := filter.Filter{
		Kinds:   []int{event.KindProfileMetadata},
		Authors: []string{pub},
		Limit:   1,
	}

	evs := cfg.Events(f)
	if len(evs) == 0 {
		return errors.New("cannot find user")
	}

	if j {
		fmt.Fprintln(os.Stdout, evs[0].Content)
		return nil
	}
	var profile Profile
	err := json.Unmarshal([]byte(evs[0].Content), &profile)
	if err != nil {
		return err
	}
	npub, err := nip19.EncodePublicKey(pub)
	if err != nil {
		return err
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
