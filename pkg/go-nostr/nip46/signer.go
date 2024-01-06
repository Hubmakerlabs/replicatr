package nip46

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip04"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/mailru/easyjson"
	"golang.org/x/exp/slices"
)

type Request struct {
	ID     string   `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Response struct {
	ID     string `json:"id"`
	Error  string `json:"error,omitempty"`
	Result string `json:"result,omitempty"`
}

type Session struct {
	SharedKey []byte
}

func (s Session) ParseRequest(evt *event.T) (Request, error) {
	var req Request

	plain, e := nip04.Decrypt(evt.Content, s.SharedKey)
	if e != nil {
		return req, fmt.Errorf("failed to decrypt evt from %s: %w", evt.PubKey, e)
	}

	e = json.Unmarshal([]byte(plain), &req)
	return req, e
}

func (s Session) MakeResponse(
	id string,
	requester string,
	result string,
	e error,
) (resp Response, evt event.T, error error) {
	if e != nil {
		resp = Response{
			ID:    id,
			Error: e.Error(),
		}
	} else if result != "" {
		resp = Response{
			ID:     id,
			Result: result,
		}
	}

	jresp, _ := json.Marshal(resp)
	ciphertext, e := nip04.Encrypt(string(jresp), s.SharedKey)
	if e != nil {
		return resp, evt, fmt.Errorf("failed to encrypt result: %w", e)
	}
	evt.Content = ciphertext

	evt.CreatedAt = timestamp.Now()
	evt.Kind = event.KindNostrConnect
	evt.Tags = tags.Tags{tags.Tag{"p", requester}}

	return resp, evt, nil
}

type Signer struct {
	secretKey string

	sessionKeys []string
	sessions    []Session

	RelaysToAdvertise map[string]relayReadWrite
}

type relayReadWrite struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

func NewSigner(secretKey string) Signer {
	return Signer{secretKey: secretKey}
}

func (p *Signer) AddRelayToAdvertise(url string, read bool, write bool) {
	p.RelaysToAdvertise[url] = relayReadWrite{read, write}
}

func (p *Signer) GetSession(clientPubkey string) (Session, error) {
	idx, exists := slices.BinarySearch(p.sessionKeys, clientPubkey)
	if exists {
		return p.sessions[idx], nil
	}

	shared, e := nip04.ComputeSharedSecret(clientPubkey, p.secretKey)
	if e != nil {
		return Session{}, fmt.Errorf("failed to compute shared secret: %w", e)
	}

	session := Session{
		SharedKey: shared,
	}

	// add to pool
	p.sessionKeys = append(p.sessionKeys, "") // bogus append just to increase the capacity
	p.sessions = append(p.sessions, Session{})
	copy(p.sessionKeys[idx+1:], p.sessionKeys[idx:])
	copy(p.sessions[idx+1:], p.sessions[idx:])
	p.sessionKeys[idx] = clientPubkey
	p.sessions[idx] = session

	return session, nil
}

func (p *Signer) HandleRequest(evt *event.T) (req Request, resp Response, eventResponse event.T, harmless bool, e error) {
	if evt.Kind != event.KindNostrConnect {
		return req, resp, eventResponse, false,
			fmt.Errorf("evt kind is %d, but we expected %d", evt.Kind, event.KindNostrConnect)
	}

	session, e := p.GetSession(evt.PubKey)
	if e != nil {
		return req, resp, eventResponse, false, e
	}

	req, e = session.ParseRequest(evt)
	if e != nil {
		return req, resp, eventResponse, false, fmt.Errorf("error parsing request: %w", e)
	}

	var result string
	var resultErr error

	switch req.Method {
	case "connect":
		result = "ack"
		harmless = true
	case "get_public_key":
		pubkey, e := keys.GetPublicKey(p.secretKey)
		if e != nil {
			resultErr = fmt.Errorf("failed to derive public key: %w", e)
			break
		} else {
			result = pubkey
			harmless = true
		}
	case "sign_event":
		if len(req.Params) != 1 {
			resultErr = fmt.Errorf("wrong number of arguments to 'sign_event'")
			break
		}
		evt := event.T{}
		e = easyjson.Unmarshal([]byte(req.Params[0]), &evt)
		if e != nil {
			resultErr = fmt.Errorf("failed to decode evt/2: %w", e)
			break
		}
		e = evt.Sign(p.secretKey)
		if e != nil {
			resultErr = fmt.Errorf("failed to sign evt: %w", e)
			break
		}
		jrevt, _ := easyjson.Marshal(evt)
		result = string(jrevt)
	case "get_relays":
		jrelays, _ := json.Marshal(p.RelaysToAdvertise)
		result = string(jrelays)
		harmless = true
	case "nip04_encrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip04_encrypt'")
			break
		}
		thirdPartyPubkey := req.Params[0]
		if !keys.IsValidPublicKeyHex(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_encrypt' is not a pubkey string")
			break
		}
		plaintext := req.Params[1]
		sharedSecret, e := nip04.ComputeSharedSecret(thirdPartyPubkey, p.secretKey)
		if e != nil {
			resultErr = fmt.Errorf("failed to compute shared secret: %w", e)
			break
		}
		ciphertext, e := nip04.Encrypt(plaintext, sharedSecret)
		if e != nil {
			resultErr = fmt.Errorf("failed to encrypt: %w", e)
			break
		}
		result = ciphertext
	case "nip04_decrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip04_decrypt'")
			break
		}
		thirdPartyPubkey := req.Params[0]
		if !keys.IsValidPublicKeyHex(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_decrypt' is not a pubkey string")
			break
		}
		ciphertext := req.Params[1]
		sharedSecret, e := nip04.ComputeSharedSecret(thirdPartyPubkey, p.secretKey)
		if e != nil {
			resultErr = fmt.Errorf("failed to compute shared secret: %w", e)
			break
		}
		plaintext, e := nip04.Decrypt(ciphertext, sharedSecret)
		if e != nil {
			resultErr = fmt.Errorf("failed to encrypt: %w", e)
			break
		}
		result = plaintext
	default:
		return req, resp, eventResponse, false,
			fmt.Errorf("unknown method '%s'", req.Method)
	}

	resp, eventResponse, e = session.MakeResponse(req.ID, evt.PubKey, result, resultErr)
	if e != nil {
		return req, resp, eventResponse, harmless, e
	}

	e = eventResponse.Sign(p.secretKey)
	if e != nil {
		return req, resp, eventResponse, harmless, e
	}

	return req, resp, eventResponse, harmless, e
}
