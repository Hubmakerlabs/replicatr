package relay

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	btcec "mleku.online/git/ec"

	"golang.org/x/net/websocket"
)

func TestPublish(t *testing.T) {
	// test note to be sent over websocket
	priv, pub := makeKeyPair(t)
	textNote := &event.T{
		Kind:      kind.TextNote,
		Content:   "hello",
		CreatedAt: timestamp.T(1672068534), // random fixed timestamp
		Tags:      tags.T{[]string{"foo", "bar"}},
		PubKey:    pub,
	}
	if err := textNote.Sign(priv); err != nil {
		t.Fatalf("textNote.Sign: %v", err)
	}

	// fake relay server
	var mu sync.Mutex // guards published to satisfy go test -race
	var published bool
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		mu.Lock()
		published = true
		mu.Unlock()
		// verify the client sent exactly the textNote
		var raw []json.RawMessage
		if err := websocket.JSON.Receive(conn, &raw); err != nil {
			t.Errorf("websocket.JSON.Receive: %v", err)
		}
		event := parseEventMessage(t, raw)
		if !bytes.Equal(event.Serialize(), textNote.Serialize()) {
			t.Errorf("received event:\n%+v\nwant:\n%+v", event, textNote)
		}
		// send back an ok nip-20 command result
		res := []any{"OK", textNote.ID, true, ""}
		if err := websocket.JSON.Send(conn, res); err != nil {
			t.Errorf("websocket.JSON.Send: %v", err)
		}
	})
	defer ws.Close()

	// connect a client and send the text note
	rl := MustRelayConnect(ws.URL)
	status, _ := rl.Publish(context.Background(), textNote)
	if status != PublishStatusSucceeded {
		t.Errorf("published status is %d, not %d", status, PublishStatusSucceeded)
	}

	if !published {
		t.Errorf("fake relay server saw no event")
	}
}

func TestPublishBlocked(t *testing.T) {
	// test note to be sent over websocket
	textNote := &event.T{Kind: kind.TextNote, Content: "hello"}
	textNote.ID = textNote.GetID()

	// fake relay server
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		// discard received message; not interested
		var raw []json.RawMessage
		if err := websocket.JSON.Receive(conn, &raw); err != nil {
			t.Errorf("websocket.JSON.Receive: %v", err)
		}
		// send back a not ok nip-20 command result
		res := []any{"OK", textNote.ID, false, "blocked"}
		websocket.JSON.Send(conn, res)
	})
	defer ws.Close()

	// connect a client and send a text note
	rl := MustRelayConnect(ws.URL)
	status, _ := rl.Publish(context.Background(), textNote)
	if status != PublishStatusFailed {
		t.Errorf("published status is %d, not %d", status, PublishStatusFailed)
	}
}

func TestPublishWriteFailed(t *testing.T) {
	// test note to be sent over websocket
	textNote := &event.T{Kind: kind.TextNote, Content: "hello"}
	textNote.ID = textNote.GetID()

	// fake relay server
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		// reject receive - force send error
		conn.Close()
	})
	defer ws.Close()

	// connect a client and send a text note
	rl := MustRelayConnect(ws.URL)
	// Force brief period of time so that publish always fails on closed socket.
	time.Sleep(1 * time.Millisecond)
	status, err := rl.Publish(context.Background(), textNote)
	if status != PublishStatusFailed {
		t.Errorf("published status is %d, not %d, err: %v", status, PublishStatusFailed, err)
	}
}

func TestConnectContext(t *testing.T) {
	// fake relay server
	var mu sync.Mutex // guards connected to satisfy go test -race
	var connected bool
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		mu.Lock()
		connected = true
		mu.Unlock()
		io.ReadAll(conn) // discard all input
	})
	defer ws.Close()

	// relay client
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r, err := RelayConnect(ctx, ws.URL)
	if err != nil {
		t.Fatalf("RelayConnectContext: %v", err)
	}
	defer r.Close()

	mu.Lock()
	defer mu.Unlock()
	if !connected {
		t.Error("fake relay server saw no client connect")
	}
}

func TestConnectContextCanceled(t *testing.T) {
	// fake relay server
	ws := newWebsocketServer(discardingHandler)
	defer ws.Close()

	// relay client
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // make ctx expired
	_, err := RelayConnect(ctx, ws.URL)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("RelayConnectContext returned %v error; want context.Canceled", err)
	}
}

func TestConnectWithOrigin(t *testing.T) {
	// fake relay server
	// default handler requires origin golang.org/x/net/websocket
	ws := httptest.NewServer(websocket.Handler(discardingHandler))
	defer ws.Close()

	// relay client
	r := NewRelay(context.Background(), normalize.URL(ws.URL))
	r.RequestHeader = http.Header{"origin": {"https://example.com"}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.Connect(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func discardingHandler(conn *websocket.Conn) {
	io.ReadAll(conn) // discard all input
}

func newWebsocketServer(handler func(*websocket.Conn)) *httptest.Server {
	return httptest.NewServer(&websocket.Server{
		Handshake: anyOriginHandshake,
		Handler:   handler,
	})
}

// anyOriginHandshake is an alternative to default in golang.org/x/net/websocket
// which checks for origin. nostr client sends no origin and it makes no difference
// for the tests here anyway.
var anyOriginHandshake = func(conf *websocket.Config, r *http.Request) (e error) {
	return nil
}

func makeKeyPair(t *testing.T) (priv, pub string) {
	t.Helper()
	privkey := GeneratePrivateKey()
	pubkey, err := nip19.GetPublicKey(privkey)
	if err != nil {
		t.Fatalf("GetPublicKey(%q): %v", privkey, err)
	}
	return privkey, pubkey
}

func MustRelayConnect(url string) *Relay {
	rl, err := RelayConnect(context.Background(), url)
	if err != nil {
		panic(err.Error())
	}
	return rl
}

func parseEventMessage(t *testing.T, raw []json.RawMessage) (evt *event.T) {
	t.Helper()
	if len(raw) < 2 {
		t.Fatalf("len(raw) = %d; want at least 2", len(raw))
	}
	var typ string
	json.Unmarshal(raw[0], &typ)
	if typ != "EVENT" {
		t.Errorf("typ = %q; want EVENT", typ)
	}
	evt = &event.T{}
	if err := json.Unmarshal(raw[1], evt); err != nil {
		t.Errorf("json.Unmarshal(`%s`): %v", string(raw[1]), err)
	}
	return evt
}

func parseSubscriptionMessage(t *testing.T, raw []json.RawMessage) (subid string,
	ff filters.T) {

	t.Helper()
	if len(raw) < 3 {
		t.Fatalf("len(raw) = %d; want at least 3", len(raw))
	}
	var typ string
	log.D.Chk(json.Unmarshal(raw[0], &typ))
	if typ != "REQ" {
		t.Errorf("typ = %q; want REQ", typ)
	}
	var id string
	if err := json.Unmarshal(raw[1], &id); err != nil {
		t.Errorf("json.Unmarshal sub id: %v", err)
	}
	for i, b := range raw[2:] {
		f := &filter.T{}
		if err := json.Unmarshal(b, f); err != nil {
			t.Errorf("json.Unmarshal filter %d: %v", i, err)
		}
		ff = append(ff, f)
	}
	return id, ff
}

func GeneratePrivateKey() string {
	params := btcec.S256().Params()
	one := new(big.Int).SetInt64(1)

	b := make([]byte, params.BitSize/8+8)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}

	k := new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	k.Mod(k, n)
	k.Add(k, one)

	return hex.EncodeToString(k.Bytes())
}
