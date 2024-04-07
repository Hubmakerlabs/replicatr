package khatru

import (
	"net/http"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/puzpuzpuz/xsync/v2"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/relayinfo"
)

var Version = "v0.0.1"
var Software = "https://github.com/Hubmakerlabs/replicatr"

func NewRelay() *Relay {
	return &Relay{
		Info: &relayinfo.T{
			Software: Software,
			Version:  Version,
			Nips:     []int{1, 11, 70},
		},

		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},

		clients:  xsync.NewTypedMapOf[*websocket.Conn, struct{}](PointerHasher[websocket.Conn]),
		serveMux: &http.ServeMux{},

		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     30 * time.Second,
		MaxMessageSize: 512000,
	}
}

type Relay struct {
	ServiceURL string

	RejectEvent              []func(c context.T, ev *event.T) (reject bool, msg string)
	RejectFilter             []func(c context.T, f filter.T) (reject bool, msg string)
	RejectCountFilter        []func(c context.T, f filter.T) (reject bool, msg string)
	OverwriteDeletionOutcome []func(c context.T, ev *event.T, deletion *event.T) (acceptDeletion bool,
		msg string)
	OverwriteResponseEvent    []func(c context.T, ev *event.T)
	OverwriteFilter           []func(c context.T, f *filter.T)
	OverwriteCountFilter      []func(c context.T, f *filter.T)
	OverwriteRelayInformation []func(c context.T, r *http.Request, info *relayinfo.T) *relayinfo.T
	StoreEvent                []func(c context.T, ev *event.T) error
	DeleteEvent               []func(c context.T, ev *event.T) error
	QueryEvents               []func(c context.T, f *filter.T) (event.C, error)
	CountEvents               []func(c context.T, f *filter.T) (int, error)
	OnConnect                 []func(c context.T)
	OnDisconnect              []func(c context.T)
	OnEventSaved              []func(c context.T, ev *event.T)
	OnEphemeralEvent          []func(c context.T, ev *event.T)

	// editing info will affect
	Info *relayinfo.T

	// for establishing websockets
	upgrader websocket.Upgrader

	// keep a connection reference to all connected clients for Server.Shutdown
	clients *xsync.MapOf[*websocket.Conn, struct{}]

	// in case you call Server.Start
	Addr       string
	serveMux   *http.ServeMux
	httpServer *http.Server

	// websocket options
	WriteWait      time.Duration // Time allowed to write a message to the peer.
	PongWait       time.Duration // Time allowed to read the next pong message from the peer.
	PingPeriod     time.Duration // Send pings to peer with this period. Must be less than pongWait.
	MaxMessageSize int64         // Maximum message size allowed from peer.
}
