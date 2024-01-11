package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr-sdk"
	"github.com/urfave/cli/v2"
)

func doProfile(cCtx *cli.Context) (e error) {
	user, j := cCtx.String("u"), cCtx.Bool("json")
	cfg := cCtx.App.Metadata["config"].(*C)
	var rl *relays.Relay
	if rl = cfg.FindRelay(context.Bg(), rp); rl == nil {
		return errors.New("cannot connect relays")
	}
	defer log.E.Chk(rl.Close())
	var pub string
	if user == "" {
		if pub, _, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
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
	f := filter.T{
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
	e = json.Unmarshal([]byte(evs[0].Content), &profile)
	if log.Fail(e) {
		return e
	}
	npub, e := nip19.EncodePublicKey(pub)
	if log.Fail(e) {
		return e
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
