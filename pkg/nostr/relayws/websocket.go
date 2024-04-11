package relayws

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"

	"mleku.dev/git/nostr/envelopes/eventenvelope"
	"mleku.dev/git/nostr/envelopes/labels"
	"mleku.dev/git/nostr/kind"

	"github.com/fasthttp/websocket"
	"mleku.dev/git/atomic"
	"mleku.dev/git/nostr/interfaces/enveloper"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type MessageType int

// The message types are defined in RFC 6455, section 11.8.
//
// Repeating here for shorter names.
const (
	// TextMessage denotes a text data message. The text message payload is
	// interpreted as UTF-8 encoded text data.
	TextMessage MessageType = websocket.TextMessage

	// BinaryMessage denotes a binary data message.
	BinaryMessage MessageType = websocket.BinaryMessage

	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text. Use the FormatCloseMessage
	// function to format a close message payload.
	CloseMessage MessageType = websocket.CloseMessage

	// PingMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PingMessage MessageType = websocket.PingMessage

	// PongMessage denotes a pong control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage MessageType = websocket.PongMessage
)

// WebSocket is a wrapper around a fasthttp/websocket with mutex locking and
// NIP-42 Auth support
type WebSocket struct {
	Conn         *websocket.Conn
	remote       atomic.String
	mutex        sync.Mutex
	Request      *http.Request // original request
	challenge    atomic.String // nip42
	Pending      atomic.Value  // for DM CLI authentication
	authPubKey   atomic.String
	Authed       chan struct{}
	OffenseCount atomic.Uint32 // when client does dumb stuff, increment this
}

// WriteMessage writes a message with a given websocket type specifier
func (ws *WebSocket) WriteMessage(t MessageType, b []byte) (err error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	if len(b) != 0 {
		log.T.F("sending message to %s %s\n%s", ws.RealRemote(), ws.AuthPubKey(), string(b))
	}
	chk.E(ws.Conn.WriteMessage(int(t), b))
	return
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
	log.T.F("sending message to %s %s %s\n%s\n%s",
		ws.RealRemote(),
		ws.AuthPubKey(),
		evkind,
		env.ToArray().String(),
		loc)
	chk.E(ws.Conn.WriteMessage(int(TextMessage), env.ToArray().Bytes()))
	return
}

const ChallengeLength = 16

// GenerateChallenge gathers new entropy to generate a new challenge.
func (ws *WebSocket) GenerateChallenge() (challenge string) {
	var err error
	// create a new challenge for this connection
	challengeBytes := make([]byte, ChallengeLength)
	if _, err = rand.Read(challengeBytes); chk.E(err) {
		// i never know what to do for this case, panic? usually
		// just ignore, it should never happen
	}
	challenge = base64.StdEncoding.EncodeToString(challengeBytes)
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
