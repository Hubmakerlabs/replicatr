package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr-sdk"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/urfave/cli/v2"
)

func Profile(cCtx *cli.Context) (e error) {
	user, j := cCtx.String("u"), cCtx.Bool("json")
	cfg := cCtx.App.Metadata["config"].(*C)
	var rl *relay.Relay
	if rl = cfg.FindRelay(context.Bg(), readPerms); rl == nil {
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
		Kinds:   kinds.T{kind.ProfileMetadata},
		Authors: []string{pub},
		Limit:   1,
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
	e = json.Unmarshal([]byte(evs[0].Content), &p)
	if log.Fail(e) {
		return e
	}
	var npub string
	if npub, e = nip19.EncodePublicKey(pub); log.Fail(e) {
		return e
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
