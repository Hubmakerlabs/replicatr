package main

import (
	"encoding/json"
	"errors"
	"fmt"

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
		fmt.Println(evs[0].Content)
		return nil
	}
	var p Profile
	e = json.Unmarshal([]byte(evs[0].Content), &p)
	if log.Fail(e) {
		return e
	}
	npub, e := nip19.EncodePublicKey(pub)
	if log.Fail(e) {
		return e
	}
	fmt.Printf("Pubkey: %v\n"+
		"Name: %v\n"+
		"DisplayName: %v\n"+
		"WebSite: %v\n"+
		"Picture: %v\n"+
		"NIP-05: %v\n"+
		"LUD-16: %v\n"+
		"About:\n%v\n",
		npub,
		p.Name,
		p.DisplayName,
		p.Website,
		p.Picture,
		p.Nip05,
		p.Lud16,
		p.About)
	return nil
}
