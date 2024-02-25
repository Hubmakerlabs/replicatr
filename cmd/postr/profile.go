package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"
	"mleku.dev/git/nostr/bech32encoding"
	"mleku.dev/git/nostr/client"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/kinds"
	"mleku.dev/git/nostr/sdk"
)

func Profile(cCtx *cli.Context) (err error) {
	user, j := cCtx.String("u"), cCtx.Bool("json")
	cfg := cCtx.App.Metadata["config"].(*C)
	var rl *client.T
	if rl = cfg.FindRelay(context.Bg(), readPerms); rl == nil {
		return errors.New("cannot connect relays")
	}
	defer chk.E(rl.Close())
	var pub string
	if user == "" {
		if pub, _, err = getPubFromSec(cfg.SecretKey); chk.D(err) {
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
		Kinds:   kinds.T{kind.ProfileMetadata},
		Authors: []string{pub},
		Limit:   &one,
	}
	evs := cfg.Events(f)
	if len(evs) == 0 {
		return errors.New("cannot find user")
	}
	log.D.S(evs[0].Content)
	if j {
		fmt.Println(evs[0].Content)
		return nil
	}
	var p Metadata
	err = json.Unmarshal([]byte(evs[0].Content), &p)
	if chk.D(err) {
		return err
	}
	var npub string
	if npub, err = bech32encoding.EncodePublicKey(pub); chk.D(err) {
		return err
	}
	fmt.Printf(
		"Name:\n\t%v\n"+
			"Pubkey:\n\t%v\n"+
			"DisplayName:\n\t%v\n"+
			"WebSite:\n\t%v\n"+
			"Picture:\n\t%v\n"+
			"Banner:\n\t%v\n"+
			"NIP-05:\n\t%v\n"+
			"LUD-16:\n\t%v\n"+
			"About:\n\t%v\n",
		p.Name,
		npub,
		p.DisplayName,
		p.Website,
		p.Picture,
		p.Banner,
		p.Nip05,
		p.Lud16,
		p.About)
	return nil
}
