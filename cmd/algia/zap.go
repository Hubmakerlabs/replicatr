package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/pointers"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/mdp/qrterminal/v3"
	"github.com/urfave/cli/v2"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip04"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
)

// Lnurlp is
type Lnurlp struct {
	Callback       string `json:"callback"`
	MaxSendable    int64  `json:"maxSendable"`
	MinSendable    int    `json:"minSendable"`
	Metadata       string `json:"metadata"`
	CommentAllowed int    `json:"commentAllowed"`
	Tag            string `json:"tag"`
	AllowsNostr    bool   `json:"allowsNostr"`
	NostrPubkey    string `json:"nostrPubkey"`
}

// Invoice is
type Invoice struct {
	PR string `json:"pr"`
}

// PayRequest is
type PayRequest struct {
	Method string `json:"method"`
	Params struct {
		Invoice string   `json:"invoice"`
		Routes  []string `json:"routes:"`
	} `json:"params"`
}

// PayResponse is
type PayResponse struct {
	ResultType *string `json:"result_type"`
	Err        *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result *struct {
		Preimage string `json:"preimage"`
	} `json:"result"`
}

func pay(cfg *Config, invoice string) (e error) {
	uri, e := url.Parse(cfg.NwcURI)
	if e != nil {
		return e
	}
	wallet := uri.Host
	host := uri.Query().Get("relay")
	secret := uri.Query().Get("secret")
	pub, e := keys.GetPublicKey(secret)
	if e != nil {
		return e
	}

	rl, e := relays.RelayConnect(context.Background(), host)
	if e != nil {
		return e
	}
	defer rl.Close()

	ss, e := nip04.ComputeSharedSecret(wallet, secret)
	if e != nil {
		return e
	}
	var req PayRequest
	req.Method = "pay_invoice"
	req.Params.Invoice = invoice
	b, e := json.Marshal(req)
	if e != nil {
		return e
	}
	content, e := nip04.Encrypt(string(b), ss)
	if e != nil {
		return e
	}

	ev := event.T{
		PubKey:    pub,
		CreatedAt: timestamp.Now(),
		Kind:      event.KindNWCWalletRequest,
		Tags:      tags.Tags{tags.Tag{"p", wallet}},
		Content:   content,
	}
	e = ev.Sign(secret)
	if e != nil {
		return e
	}

	filters := []filter.T{{
		Tags: filter.TagMap{
			"p": []string{pub},
			"e": []string{ev.ID},
		},
		Kinds: []int{event.KindNWCWalletInfo, event.KindNWCWalletResponse, event.KindNWCWalletRequest},
		Limit: 1,
	}}
	sub, e := rl.Subscribe(context.Background(), filters)
	if e != nil {
		return e
	}

	e = rl.Publish(context.Background(), ev)
	if e != nil {
		return e
	}

	er := <-sub.Events
	content, e = nip04.Decrypt(er.Content, ss)
	if e != nil {
		return e
	}
	var resp PayResponse
	e = json.Unmarshal([]byte(content), &resp)
	if e != nil {
		return e
	}
	if resp.Err != nil {
		return fmt.Errorf(resp.Err.Message)
	}
	json.NewEncoder(os.Stdout).Encode(resp)
	return nil
}

// ZapInfo is
func (cfg *Config) ZapInfo(pub string) (*Lnurlp, error) {
	rl := cfg.FindRelay(context.Background(), RelayPerms{Read: true})
	if rl == nil {
		return nil, errors.New("cannot connect relays")
	}
	defer rl.Close()

	// get set-metadata
	f := filter.T{
		Kinds:   []int{event.KindProfileMetadata},
		Authors: []string{pub},
		Limit:   1,
	}

	evs := cfg.Events(f)
	if len(evs) == 0 {
		return nil, errors.New("cannot find user")
	}

	var profile Profile
	e := json.Unmarshal([]byte(evs[0].Content), &profile)
	if e != nil {
		return nil, e
	}

	tok := strings.SplitN(profile.Lud16, "@", 2)
	if e != nil {
		return nil, e
	}
	if len(tok) != 2 {
		return nil, errors.New("receipt address is not valid")
	}

	resp, e := http.Get("https://" + tok[1] + "/.well-known/lnurlp/" + tok[0])
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()

	var lp Lnurlp
	e = json.NewDecoder(resp.Body).Decode(&lp)
	if e != nil {
		return nil, e
	}
	return &lp, nil
}

func doZap(cCtx *cli.Context) (e error) {
	amount := cCtx.Uint64("amount")
	comment := cCtx.String("comment")
	if cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}

	if cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}

	cfg := cCtx.App.Metadata["config"].(*Config)

	var sk string
	if _, s, e := nip19.Decode(cfg.PrivateKey); e == nil {
		sk = s.(string)
	} else {
		return e
	}

	receipt := ""
	zr := event.T{}
	zr.Tags = tags.Tags{}

	if pub, e := keys.GetPublicKey(sk); e == nil {
		if _, e := nip19.EncodePublicKey(pub); e != nil {
			return e
		}
		zr.PubKey = pub
	} else {
		return e
	}

	zr.Tags = zr.Tags.AppendUnique(tags.Tag{"amount", fmt.Sprint(amount * 1000)})
	relays := tags.Tag{"relays"}
	for k, v := range cfg.Relays {
		if v.Write {
			relays = append(relays, k)
		}
	}
	zr.Tags = zr.Tags.AppendUnique(relays)
	if prefix, s, e := nip19.Decode(cCtx.Args().First()); e == nil {
		switch prefix {
		case "nevent":
			receipt = s.(pointers.EventPointer).Author
			zr.Tags = zr.Tags.AppendUnique(tags.Tag{"p", receipt})
			zr.Tags = zr.Tags.AppendUnique(tags.Tag{"e", s.(pointers.EventPointer).ID})
		case "note":
			evs := cfg.Events(filter.T{IDs: []string{s.(string)}})
			if len(evs) != 0 {
				receipt = evs[0].PubKey
				zr.Tags = zr.Tags.AppendUnique(tags.Tag{"p", receipt})
			}
			zr.Tags = zr.Tags.AppendUnique(tags.Tag{"e", s.(string)})
		case "npub":
			receipt = s.(string)
			zr.Tags = zr.Tags.AppendUnique(tags.Tag{"p", receipt})
		default:
			return errors.New("invalid argument")
		}
	}

	zr.Kind = event.KindZapRequest // 9734
	zr.CreatedAt = timestamp.Now()
	zr.Content = comment
	if e := zr.Sign(sk); e != nil {
		return e
	}
	b, e := zr.MarshalJSON()
	if e != nil {
		return e
	}

	zi, e := cfg.ZapInfo(receipt)
	if e != nil {
		return e
	}
	u, e := url.Parse(zi.Callback)
	if e != nil {
		return e
	}
	param := url.Values{}
	param.Set("amount", fmt.Sprint(amount*1000))
	param.Set("nostr", string(b))
	u.RawQuery = param.Encode()
	resp, e := http.Get(u.String())
	if e != nil {
		return e
	}
	defer resp.Body.Close()

	var iv Invoice
	e = json.NewDecoder(resp.Body).Decode(&iv)
	if e != nil {
		return e
	}

	if cfg.NwcURI == "" {
		config := qrterminal.Config{
			HalfBlocks: false,
			Level:      qrterminal.L,
			Writer:     os.Stdout,
			WhiteChar:  qrterminal.WHITE,
			BlackChar:  qrterminal.BLACK,
			QuietZone:  2,
			WithSixel:  true,
		}
		fmt.Println("lightning:" + iv.PR)
		qrterminal.GenerateWithConfig("lightning:"+iv.PR, config)
	} else {
		pay(cCtx.App.Metadata["config"].(*Config), iv.PR)
	}
	return nil
}
