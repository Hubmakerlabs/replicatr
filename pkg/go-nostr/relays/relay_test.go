package relays

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/normalize"
	"golang.org/x/net/websocket"
)

func TestPublish(t *testing.T) {
	// test note to be sent over websocket
	priv, pub := makeKeyPair(t)
	textNote := event.T{
		Kind:      event.KindTextNote,
		Content:   "hello",
		CreatedAt: timestamp.T(1672068534), // random fixed timestamp
		Tags:      tags.Tags{[]string{"foo", "bar"}},
		PubKey:    pub,
	}
	if e := textNote.Sign(priv); e != nil {
		t.Fatalf("textNote.Sign: %v", e)
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
		if e := websocket.JSON.Receive(conn, &raw); e != nil {
			t.Errorf("websocket.JSON.Receive: %v", e)
		}
		event := parseEventMessage(t, raw)
		if !bytes.Equal(event.Serialize(), textNote.Serialize()) {
			t.Errorf("received event:\n%+v\nwant:\n%+v", event, textNote)
		}
		// send back an ok nip-20 command result
		res := []any{"OK", textNote.ID, true, ""}
		if e := websocket.JSON.Send(conn, res); e != nil {
			t.Errorf("websocket.JSON.Send: %v", e)
		}
	})
	defer ws.Close()

	// connect a client and send the text note
	rl := MustRelayConnect(ws.URL)
	e := rl.Publish(context.Background(), textNote)
	if e != nil {
		t.Errorf("publish should have succeeded")
	}
	if !published {
		t.Errorf("fake relay server saw no event")
	}
}

func TestPublishBlocked(t *testing.T) {
	// test note to be sent over websocket
	textNote := event.T{Kind: event.KindTextNote, Content: "hello"}
	textNote.ID = textNote.GetID()

	// fake relay server
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		// discard received message; not interested
		var raw []json.RawMessage
		if e := websocket.JSON.Receive(conn, &raw); e != nil {
			t.Errorf("websocket.JSON.Receive: %v", e)
		}
		// send back a not ok nip-20 command result
		res := []any{"OK", textNote.ID, false, "blocked"}
		websocket.JSON.Send(conn, res)
	})
	defer ws.Close()

	// connect a client and send a text note
	rl := MustRelayConnect(ws.URL)
	e := rl.Publish(context.Background(), textNote)
	if e == nil {
		t.Errorf("should have failed to publish")
	}
}

func TestPublishWriteFailed(t *testing.T) {
	// test note to be sent over websocket
	textNote := event.T{Kind: event.KindTextNote, Content: "hello"}
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
	e := rl.Publish(context.Background(), textNote)
	if e == nil {
		t.Errorf("should have failed to publish")
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
	r, e := RelayConnect(ctx, ws.URL)
	if e != nil {
		t.Fatalf("RelayConnectContext: %v", e)
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
	_, e := RelayConnect(ctx, ws.URL)
	if !errors.Is(e, context.Canceled) {
		t.Errorf("RelayConnectContext returned %v error; want context.Canceled", e)
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
	e := r.Connect(ctx)
	if e != nil {
		t.Errorf("unexpected error: %v", e)
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
var anyOriginHandshake = func(conf *websocket.Config, r *http.Request) error {
	return nil
}

func makeKeyPair(t *testing.T) (priv, pub string) {
	t.Helper()
	privkey := keys.GeneratePrivateKey()
	pubkey, e := keys.GetPublicKey(privkey)
	if e != nil {
		t.Fatalf("GetPublicKey(%q): %v", privkey, e)
	}
	return privkey, pubkey
}

func parseEventMessage(t *testing.T, raw []json.RawMessage) event.T {
	t.Helper()
	if len(raw) < 2 {
		t.Fatalf("len(raw) = %d; want at least 2", len(raw))
	}
	var typ string
	json.Unmarshal(raw[0], &typ)
	if typ != "EVENT" {
		t.Errorf("typ = %q; want EVENT", typ)
	}
	var evt event.T
	if e := json.Unmarshal(raw[1], &evt); e != nil {
		t.Errorf("json.Unmarshal(`%s`): %v", string(raw[1]), e)
	}
	return evt
}

func parseSubscriptionMessage(t *testing.T, raw []json.RawMessage) (subid string, filters []filter.T) {
	t.Helper()
	if len(raw) < 3 {
		t.Fatalf("len(raw) = %d; want at least 3", len(raw))
	}
	var typ string
	json.Unmarshal(raw[0], &typ)
	if typ != "REQ" {
		t.Errorf("typ = %q; want REQ", typ)
	}
	var id string
	if e := json.Unmarshal(raw[1], &id); e != nil {
		t.Errorf("json.Unmarshal sub id: %v", e)
	}
	var ff []filter.T
	for i, b := range raw[2:] {
		var f filter.T
		if e := json.Unmarshal(b, &f); e != nil {
			t.Errorf("json.Unmarshal filter %d: %v", i, e)
		}
		ff = append(ff, f)
	}
	return id, ff
}
