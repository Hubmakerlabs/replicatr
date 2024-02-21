package relayws

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/labels"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/fasthttp/websocket"
	"mleku.dev/git/atomic"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

// WebSocket is a wrapper around a fasthttp/websocket with mutex locking and
// NIP-42 Auth support
type WebSocket struct {
	Conn       *websocket.Conn
	remote     atomic.String
	mutex      sync.Mutex
	Request    *http.Request // original request
	challenge  atomic.String // nip42
	Pending    atomic.Value  // for DM CLI authentication
	authPubKey atomic.String
	Authed     chan struct{}
}

// WriteMessage writes a message with a given websocket type specifier
func (ws *WebSocket) WriteMessage(t int, b []byte) (err error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	if len(b) != 0 {
		log.T.F("sending message to %s %s\n%s", ws.RealRemote(), ws.AuthPubKey(), string(b))
	}
	return ws.Conn.WriteMessage(t, b)
}

// WriteEnvelope writes a message with a given websocket type specifier
func (ws *WebSocket) WriteEnvelope(env enveloper.I) (err error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	var file string
	var line int
	_, file, line, _ = runtime.Caller(1)
	loc := fmt.Sprintf("%s:%d", file, line)
	var evkind string
	var ek kind.T
	if env.Label() == labels.EVENT {
		ek = env.(*eventenvelope.T).Event.Kind
		evkind = kind.GetString(ek)
	}
	log.D.F("sending message to %s %s %s\n%s\n%s",
		ws.RealRemote(),
		ws.AuthPubKey(),
		evkind,
		env.ToArray().String(),
		loc)
	return ws.Conn.WriteMessage(websocket.TextMessage, env.Bytes())
}

const ChallengeLength = 32

// GenerateChallenge gathers new entropy to generate a new challenge.
func (ws *WebSocket) GenerateChallenge() (challenge string) {
	var err error
	// create a new challenge for this connection
	challengeBytes := make([]byte, ChallengeLength)
	if _, err = rand.Read(challengeBytes); chk.E(err) {
		// i never know what to do for this case, panic? usually
		// just ignore, it should never happen
	}
	challenge = hex.EncodeToString(challengeBytes)
	ws.challenge.Store(challenge)
	return
}

// Challenge returns the current challenge on a websocket.
func (ws *WebSocket) Challenge() (challenge string) { return ws.challenge.Load() }

// RealRemote returns the current real remote.
func (ws *WebSocket) RealRemote() (remote string) { return ws.remote.Load() }
func (ws *WebSocket) SetRealRemote(remote string) { ws.remote.Store(remote) }

// AuthPubKey returns the current authed Pubkey.
func (ws *WebSocket) AuthPubKey() (a string) { return ws.authPubKey.Load() }
func (ws *WebSocket) SetAuthPubKey(a string) { ws.authPubKey.Store(a) }
