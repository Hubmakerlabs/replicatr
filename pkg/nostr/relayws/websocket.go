package relayws

import (
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
	"mleku.online/git/atomic"
	"mleku.online/git/slog"
)

var log, chk = slog.New(os.Stderr)

// WebSocket is a wrapper around a fasthttp/websocket with mutex locking and
// NIP-42 Auth support
type WebSocket struct {
	Conn       *websocket.Conn
	RealRemote atomic.String
	mutex      sync.Mutex
	Request    *http.Request // original request
	Challenge  atomic.String // nip42
	Pending    atomic.Value  // for DM CLI authentication
	AuthPubKey atomic.String
	Authed     chan struct{}
}

// WriteMessage writes a message with a given websocket type specifier
func (ws *WebSocket) WriteMessage(t int, b []byte) (err error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	// if slog.GetLogLevel() >= slog.Trace && len(b) == 0 {
	// 	var file string
	// 	var line int
	// 	_, file, line, _ = runtime.Caller(1)
	// 	loc := fmt.Sprintf("%s:%d", file, line)
	// log.T.F("sending ping/pong to %s %s %s", ws.RealRemote.Load(), ws.AuthPubKey.Load(), loc)
	// } else
	if len(b) != 0 {
		log.T.F("sending message to %s %s\n%s", ws.RealRemote.Load(), ws.AuthPubKey.Load(), string(b))
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
	// log privileged kinds more visibly for debugging
	// if kinds.IsPrivileged(ek) {
	log.T.F("sending message to %s %s %s\n%s\n%s",
		ws.RealRemote.Load(),
		ws.AuthPubKey.Load(),
		evkind,
		env.ToArray().String(),
		loc)
	// } else {
	// log.T.F("sending message to %s %s %s\n%s\n%s",
	// 	ws.AuthPubKey.Load(), ws.AuthPubKey.Load(),
	// 	evkind, env.ToArray().String(), loc)
	// }
	return ws.Conn.WriteMessage(websocket.TextMessage, env.Bytes())
}
