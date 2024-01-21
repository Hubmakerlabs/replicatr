package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip4"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/mdp/qrterminal/v3"
	"github.com/urfave/cli/v2"
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

func pay(cfg *C, invoice string) (e error) {
	uri, e := url.Parse(cfg.NwcURI)
	if log.Fail(e) {
		return e
	}
	wallet := uri.Host
	host := uri.Query().Get("relay")
	secret := uri.Query().Get("secret")
	pub, e := keys.GetPublicKey(secret)
	if log.Fail(e) {
		return e
	}

	rl, e := relay.RelayConnect(context.Bg(), host)
	if log.Fail(e) {
		return e
	}
	defer log.Fail(rl.Close())

	ss, e := nip4.ComputeSharedSecret(wallet, secret)
	if log.Fail(e) {
		return e
	}
	var req PayRequest
	req.Method = "pay_invoice"
	req.Params.Invoice = invoice
	b, e := json.Marshal(req)
	if log.Fail(e) {
		return e
	}
	content, e := nip4.Encrypt(string(b), ss)
	if log.Fail(e) {
		return e
	}

	ev := &event.T{
		PubKey:    pub,
		CreatedAt: timestamp.Now(),
		Kind:      kind.NWCWalletRequest,
		Tags:      tags.T{{"p", wallet}},
		Content:   content,
	}
	e = ev.Sign(secret)
	if log.Fail(e) {
		return e
	}

	f := filters.T{{
		Tags: filter.TagMap{
			"p": []string{pub},
			"e": []string{ev.ID.String()},
		},
		Kinds: kinds.T{kind.NWCWalletInfo, kind.NWCWalletResponse,
			kind.NWCWalletRequest},
		Limit: 1,
	}}
	sub, e := rl.Subscribe(context.Bg(), f)
	if log.Fail(e) {
		return e
	}

	_, e = rl.Publish(context.Bg(), ev)
	if log.Fail(e) {
		return e
	}

	er := <-sub.Events
	var c []byte
	c, e = nip4.Decrypt(er.Content, ss)
	content = string(c)
	if log.Fail(e) {
		return e
	}
	var resp PayResponse
	e = json.Unmarshal([]byte(content), &resp)
	if log.Fail(e) {
		return e
	}
	if resp.Err != nil {
		return fmt.Errorf(resp.Err.Message)
	}
	log.Fail(json.NewEncoder(os.Stdout).Encode(resp))
	return nil
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

	cfg := cCtx.App.Metadata["config"].(*C)
	var pub, sk string
	if pub, sk, e = getPubFromSec(cfg.SecretKey); log.Fail(e) {
		return
	}
	receipt := ""
	zr := event.T{
		PubKey: pub,
		Tags:   tags.T{},
	}
	zr.Tags = zr.Tags.AppendUnique(tag.T{"amount", fmt.Sprint(amount * 1000)})
	rls := tag.T{"relays"}
	for k, v := range cfg.Relays {
		if v.Write {
			rls = append(rls, k)
		}
	}
	zr.Tags = zr.Tags.AppendUnique(rls)
	var prefix string
	var s any
	if prefix, s, e = bech32encoding.Decode(cCtx.Args().First()); !log.Fail(e) {
		switch prefix {
		case "nevent":
			receipt = s.(pointers.Event).Author
			zr.Tags = zr.Tags.AppendUnique(tag.T{"p", receipt})
			zr.Tags = zr.Tags.AppendUnique(tag.T{"e",
				string(s.(pointers.Event).ID)})
		case "note":
			evs := cfg.Events(filter.T{IDs: []string{s.(string)}})
			if len(evs) != 0 {
				receipt = evs[0].PubKey
				zr.Tags = zr.Tags.AppendUnique(tag.T{"p", receipt})
			}
			zr.Tags = zr.Tags.AppendUnique(tag.T{"e", s.(string)})
		case "npub":
			receipt = s.(string)
			zr.Tags = zr.Tags.AppendUnique(tag.T{"p", receipt})
		default:
			return errors.New("invalid argument")
		}
	}
	zr.Kind = kind.ZapRequest // 9734
	zr.CreatedAt = timestamp.Now()
	zr.Content = comment
	if e = zr.Sign(sk); log.Fail(e) {
		return e
	}
	var b []byte
	if b, e = zr.MarshalJSON(); log.Fail(e) {
		return e
	}
	var zi *Lnurlp
	if zi, e = cfg.ZapInfo(receipt); log.Fail(e) {
		return e
	}
	var u *url.URL
	u, e = url.Parse(zi.Callback)
	if log.Fail(e) {
		return e
	}
	param := url.Values{}
	param.Set("amount", fmt.Sprint(amount*1000))
	param.Set("nostr", string(b))
	u.RawQuery = param.Encode()
	var resp *http.Response
	if resp, e = http.Get(u.String()); log.Fail(e) {
		return e
	}
	defer log.Fail(resp.Body.Close())
	var iv Invoice
	if e = json.NewDecoder(resp.Body).Decode(&iv); log.Fail(e) {
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
		log.Fail(pay(cCtx.App.Metadata["config"].(*C), iv.PR))
	}
	return nil
}
