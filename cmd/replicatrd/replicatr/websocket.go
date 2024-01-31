package replicatr

import (
	"net/http"
	"runtime"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/interfaces/enveloper"
	"github.com/fasthttp/websocket"
	"mleku.online/git/slog"
)

// WebSocket is a wrapper around a fasthttp/websocket with mutex locking and
// NIP-42 Auth support
type WebSocket struct {
	conn            *websocket.Conn
	RealRemote      string
	mutex           sync.Mutex
	Request         *http.Request // original request
	Challenge       string        // nip42
	AuthedPublicKey string
	Authed          chan struct{}
	authLock        sync.Mutex
}

// // WriteJSON writes an object as JSON to the websocket
// func (ws *WebSocket) WriteJSON(any any) (err error) {
// 	ws.mutex.Lock()
// 	defer ws.mutex.Unlock()
// 	return ws.conn.WriteJSON(any)
// }

// WriteMessage writes a message with a given websocket type specifier
func (ws *WebSocket) WriteMessage(t int, b []byte) (err error) {
	if len(b) == 0 {
	}
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	if slog.GetLogLevel() >= slog.Trace && len(b) == 0 {
		var file string
		var line int
		_, file, line, _ = runtime.Caller(1)
		log.T.F("sending ping/pong to %s %s:%d", ws.RealRemote, file, line)
	} else {
		log.D.F("sending message to %s\n%s", ws.RealRemote, string(b))
	}
	return ws.conn.WriteMessage(t, b)
}

// WriteEnvelope writes a message with a given websocket type specifier
func (ws *WebSocket) WriteEnvelope(env enveloper.I) (err error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	var file string
	var line int
	_, file, line, _ = runtime.Caller(1)
	log.D.F("sending message to %s\n%s\n%s:%d\n",
		ws.RealRemote, env.ToArray().String(), file, line)
	return ws.conn.WriteMessage(websocket.TextMessage, env.Bytes())
}
