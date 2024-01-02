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

	"github.com/mdp/qrterminal/v3"
	"github.com/urfave/cli/v2"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip4"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/pointers"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
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
		Invoice string `json:"invoice"`
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

func pay(cfg *clientConfig, invoice string) error {
	uri, err := url.Parse(cfg.NwcURI)
	if err != nil {
		return err
	}
	wallet := uri.Host
	host := uri.Query().Get("relay")
	secret := uri.Query().Get("secret")
	pub, err := nip19.GetPublicKey(secret)
	if err != nil {
		return err
	}
	relay, err := nostr.RelayConnect(context.Background(), host)
	if err != nil {
		return err
	}
	defer relay.Close()
	ss, err := nip4.ComputeSharedSecret(wallet, secret)
	if err != nil {
		return err
	}
	var req PayRequest
	req.Method = "pay_invoice"
	req.Params.Invoice = invoice
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	content, err := nip4.Encrypt(string(b), ss)
	if err != nil {
		return err
	}
	ev := &nip1.Event{
		PubKey:    pub,
		CreatedAt: timestamp.Now(),
		Kind:      kind.NWCWalletRequest,
		Tags:      tags.T{{"p", wallet}},
		Content:   content,
	}
	err = ev.Sign(secret)
	if err != nil {
		return err
	}
	filters := []*nip1.Filter{{
		Tags: nip1.TagMap{
			"p": []string{pub},
			"e": []string{string(ev.ID)},
		},
		Kinds: kinds.T{kind.NWCWalletInfo, kind.NWCWalletResponse, kind.NWCWalletRequest},
		Limit: 1,
	}}
	sub, err := relay.Subscribe(context.Background(), filters)
	if err != nil {
		return err
	}
	_, err = relay.Publish(context.Background(), ev)
	if err != nil {
		return err
	}
	er := <-sub.Events
	var decrypted []byte
	decrypted, err = nip4.Decrypt(er.Content, ss)
	if err != nil {
		return err
	}
	content = string(decrypted)
	var resp PayResponse
	err = json.Unmarshal([]byte(content), resp)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return fmt.Errorf(resp.Err.Message)
	}
	json.NewEncoder(os.Stdout).Encode(resp)
	return nil
}

// ZapInfo is
func (cfg *clientConfig) ZapInfo(pub string) (*Lnurlp, error) {
	relay := cfg.FindRelay(context.Background(), relayPerms{Read: true})
	if relay == nil {
		return nil, errors.New("cannot connect relays")
	}
	defer relay.Close()
	// get set-metadata
	filter := &nip1.Filter{
		Kinds:   kinds.T{kind.ProfileMetadata},
		Authors: []string{pub},
		Limit:   1,
	}
	evs := cfg.Events(filter)
	if len(evs) == 0 {
		return nil, errors.New("cannot find user")
	}
	var profile userProfile
	err := json.Unmarshal([]byte(evs[0].Content), &profile)
	if err != nil {
		return nil, err
	}
	tok := strings.SplitN(profile.Lud16, "@", 2)
	if err != nil {
		return nil, err
	}
	if len(tok) != 2 {
		return nil, errors.New("receipt address is not valid")
	}
	resp, err := http.Get("https://" + tok[1] + "/.well-known/lnurlp/" + tok[0])
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var lp Lnurlp
	err = json.NewDecoder(resp.Body).Decode(&lp)
	if err != nil {
		return nil, err
	}
	return &lp, nil
}

func doZap(cCtx *cli.Context) error {
	amount := cCtx.Uint64("amount")
	comment := cCtx.String("comment")
	if cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	if cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}
	cfg := cCtx.App.Metadata["config"].(*clientConfig)
	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	receipt := ""
	zr := &nip1.Event{}
	zr.Tags = tags.T{}
	if pub, err := nip19.GetPublicKey(sk); err == nil {
		if _, err := nip19.EncodePublicKey(pub); err != nil {
			return err
		}
		zr.PubKey = pub
	} else {
		return err
	}
	zr.Tags = zr.Tags.AppendUnique(tag.T{"amount", fmt.Sprint(amount * 1000)})
	relays := tag.T{"relays"}
	for k, v := range cfg.Relays {
		if v.Write {
			relays = append(relays, k)
		}
	}
	zr.Tags = zr.Tags.AppendUnique(relays)
	if prefix, s, err := nip19.Decode(cCtx.Args().First()); err == nil {
		switch prefix {
		case "nevent":
			receipt = s.(pointers.Event).Author
			zr.Tags = zr.Tags.AppendUnique(tag.T{"p", receipt})
			zr.Tags = zr.Tags.AppendUnique(tag.T{"e", string(s.(pointers.Event).ID)})
		case "note":
			evs := cfg.Events(&nip1.Filter{IDs: []string{s.(string)}})
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
	if err := zr.Sign(sk); err != nil {
		return err
	}
	b, err := zr.MarshalJSON()
	if err != nil {
		return err
	}
	zi, err := cfg.ZapInfo(receipt)
	if err != nil {
		return err
	}
	u, err := url.Parse(zi.Callback)
	if err != nil {
		return err
	}
	param := url.Values{}
	param.Set("amount", fmt.Sprint(amount*1000))
	param.Set("nostr", string(b))
	u.RawQuery = param.Encode()
	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var iv Invoice
	err = json.NewDecoder(resp.Body).Decode(&iv)
	if err != nil {
		return err
	}
	if cfg.NwcURI == "" {
		config := qrterminal.Config{
			HalfBlocks: false,
			Level:      qrterminal.L,
			Writer:     os.Stdout,
			WhiteChar:  qrterminal.WHITE,
			BlackChar:  qrterminal.BLACK,
			QuietZone:  2,
		}
		fmt.Println("lightning:" + iv.PR)
		qrterminal.GenerateWithConfig("lightning:"+iv.PR, config)
	} else {
		pay(cCtx.App.Metadata["config"].(*clientConfig), iv.PR)
	}
	return nil
}
