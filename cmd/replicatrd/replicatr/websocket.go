package replicatr

// WebSocket is a wrapper around a fasthttp/websocket with mutex locking and
// NIP-42 Auth support
type WebSocket struct {
	conn            *Conn
	mutex           Mutex
	Request         *Request // original request
	Challenge       string   // nip42
	AuthedPublicKey string
	Authed          chan struct{}
	authLock        Mutex
}

// WriteJSON writes an object as JSON to the websocket
func (ws *WebSocket) WriteJSON(any any) (e error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.conn.WriteJSON(any)
}

// WriteMessage writes a message with a given websocket type specifier
func (ws *WebSocket) WriteMessage(t int, b []byte) (e error) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	return ws.conn.WriteMessage(t, b)
}
