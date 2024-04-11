package bunker

import (
	"encoding/json"
	"fmt"

	"mleku.dev/git/nostr/crypt"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/tags"
	"mleku.dev/git/nostr/timestamp"
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

type Signer interface {
	GetSession(clientPubkey string) (Session, bool)
	HandleRequest(ev *event.T) (req Request, resp Response, eventResponse *event.T, harmless bool, err error)
}

type Session struct {
	SharedKey []byte
}

type RelayReadWrite struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

func (s Session) ParseRequest(event *event.T) (Request, error) {
	var req Request
	plain, err := crypt.Decrypt(event.Content, s.SharedKey)
	if err != nil {
		return req, fmt.Errorf("failed to decrypt event from %s: %w", event.PubKey, err)
	}
	err = json.Unmarshal([]byte(plain), &req)
	return req, err
}

func (s Session) MakeResponse(
	id string,
	requester string,
	result string,
	err error,
) (resp Response, evt *event.T, error error) {
	if err != nil {
		resp = Response{
			ID:    id,
			Error: err.Error(),
		}
	} else if result != "" {
		resp = Response{
			ID:     id,
			Result: result,
		}
	}
	jresp, _ := json.Marshal(resp)
	ciphertext, err := crypt.Encrypt(string(jresp), s.SharedKey)
	if err != nil {
		return resp, evt, fmt.Errorf("failed to encrypt result: %w", err)
	}
	evt.Content = ciphertext
	evt.CreatedAt = timestamp.Now()
	evt.Kind = kind.NostrConnect
	evt.Tags = tags.T{{"p", requester}}
	return resp, evt, nil
}
