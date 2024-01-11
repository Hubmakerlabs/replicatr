package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip04"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/pointers"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/relays"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
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

	rl, e := relays.RelayConnect(context.Bg(), host)
	if log.Fail(e) {
		return e
	}
	defer rl.Close()

	ss, e := nip04.ComputeSharedSecret(wallet, secret)
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
	content, e := nip04.Encrypt(string(b), ss)
	if log.Fail(e) {
		return e
	}

	ev := &event.T{
		PubKey:    pub,
		CreatedAt: timestamp.Now(),
		Kind:      event.KindNWCWalletRequest,
		Tags:      tags.Tags{tags.Tag{"p", wallet}},
		Content:   content,
	}
	e = ev.Sign(secret)
	if log.Fail(e) {
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
	sub, e := rl.Subscribe(context.Bg(), filters)
	if log.Fail(e) {
		return e
	}

	e = rl.Publish(context.Bg(), ev)
	if log.Fail(e) {
		return e
	}

	er := <-sub.Events
	content, e = nip04.Decrypt(er.Content, ss)
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
	json.NewEncoder(os.Stdout).Encode(resp)
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
		Tags:   tags.Tags{},
	}
	zr.Tags = zr.Tags.AppendUnique(tags.Tag{"amount", fmt.Sprint(amount * 1000)})
	rls := tags.Tag{"relays"}
	for k, v := range cfg.Relays {
		if v.Write {
			rls = append(rls, k)
		}
	}
	zr.Tags = zr.Tags.AppendUnique(rls)
	var prefix string
	var s any
	if prefix, s, e = nip19.Decode(cCtx.Args().First()); !log.Fail(e) {
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
