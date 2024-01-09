package relay

import (
	"net/http"
	"os"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayinfo"
	"github.com/fasthttp/websocket"
	"github.com/puzpuzpuz/xsync/v2"
)

func New() *Relay {
	return &Relay{
		Log: log2.New(os.Stderr, "", 0),

		Info: &relayinfo.T{
			Software:      "https://github.com/Hubmakerlabs/replicatr/cmd/khatru",
			Version:       "n/a",
			SupportedNIPs: make([]int, 0),
		},

		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},

		clients:  xsync.NewTypedMapOf[*websocket.Conn, struct{}](pointerHasher[websocket.Conn]),
		serveMux: &http.ServeMux{},

		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     30 * time.Second,
		MaxMessageSize: 512000,
	}
}

type Relay struct {
	ServiceURL string

	RejectEvent               []func(ctx context.T, ev *event.T) (reject bool, msg string)
	RejectFilter              []func(ctx context.T, f *filter.T) (reject bool, msg string)
	RejectCountFilter         []func(ctx context.T, f *filter.T) (reject bool, msg string)
	OverwriteDeletionOutcome  []func(ctx context.T, target *event.T, deletion *event.T) (acceptDeletion bool, msg string)
	OverwriteResponseEvent    []func(ctx context.T, ev *event.T)
	OverwriteFilter           []func(ctx context.T, f *filter.T)
	OverwriteCountFilter      []func(ctx context.T, f *filter.T)
	OverwriteRelayInformation []func(ctx context.T, r *http.Request, info *relayinfo.T) *relayinfo.T
	StoreEvent                []func(ctx context.T, ev *event.T) error
	DeleteEvent               []func(ctx context.T, ev *event.T) error
	QueryEvents               []func(ctx context.T, f *filter.T) (chan *event.T, error)
	CountEvents               []func(ctx context.T, f *filter.T) (int64, error)
	OnAuth                    []func(ctx context.T, pubkey string)
	OnConnect                 []func(ctx context.T)
	OnDisconnect              []func(ctx context.T)
	OnEventSaved              []func(ctx context.T, ev *event.T)

	// editing info will affect
	Info *relayinfo.T

	// Default logger, as set by NewServer, is a stdlib logger prefixed with "[khatru-relay] ",
	// outputting to stderr.
	*log2.Log

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
