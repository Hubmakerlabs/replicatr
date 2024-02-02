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
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/fasthttp/websocket"
	"mleku.online/git/slog"
)

var log = slog.New(os.Stderr, "nostr/relayws")

// WebSocket is a wrapper around a fasthttp/websocket with mutex locking and
// NIP-42 Auth support
type WebSocket struct {
	Conn       *websocket.Conn
	RealRemote string
	mutex      sync.Mutex
	Request    *http.Request // original request
	Challenge  string        // nip42
	AuthPubKey string
	Authed     chan struct{}
}

// WriteMessage writes a message with a given websocket type specifier
func (ws *WebSocket) WriteMessage(t int, b []byte) (err error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	if slog.GetLogLevel() >= slog.Trace && len(b) == 0 {
		var file string
		var line int
		_, file, line, _ = runtime.Caller(1)
		log.T.F("sending ping/pong to %s %s:%d", ws.RealRemote, file, line)
	} else if len(b) != 0 {
		log.D.F("sending message to %s\n%s", ws.RealRemote, string(b))
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
	var evkind string
	var ek kind.T
	if env.Label() == labels.EVENT {
		kind.MapMx.Lock()
		ek = env.(*eventenvelope.T).Event.Kind
		v, ok := kind.Map[ek]
		if ok {
			evkind = fmt.Sprintf(" (%s)", v)
		}
		kind.MapMx.Unlock()
	}
	// log privileged kinds more visibly for debugging
	if kinds.IsPrivileged(ek) {
		log.D.F("sending message to %s%s\n%s\n%s:%d\n", ws.RealRemote,
			evkind, env.ToArray().String(), file, line)
	} else {
		log.T.F("sending message to %s%s\n%s\n%s:%d\n", ws.RealRemote,
			evkind, env.ToArray().String(), file, line)
	}
	return ws.Conn.WriteMessage(websocket.TextMessage, env.Bytes())
}
