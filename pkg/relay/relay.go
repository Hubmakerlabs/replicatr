package relay

import (
	"context"
	"net/http"
	"time"

	log2 "mleku.online/git/log"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip11"
	"github.com/fasthttp/websocket"
	"github.com/puzpuzpuz/xsync/v2"
)

func New() *Relay {
	return &Relay{
		Log: log2.GetLogger(),

		Info: &nip11.RelayInfo{
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

	RejectEvent               []func(ctx context.Context, event *nip1.Event) (reject bool, msg string)
	RejectFilter              []func(ctx context.Context, filter *nip1.Filter) (reject bool, msg string)
	RejectCountFilter         []func(ctx context.Context, filter *nip1.Filter) (reject bool, msg string)
	OverwriteDeletionOutcome  []func(ctx context.Context, target *nip1.Event, deletion *nip1.Event) (acceptDeletion bool, msg string)
	OverwriteResponseEvent    []func(ctx context.Context, event *nip1.Event)
	OverwriteFilter           []func(ctx context.Context, filter *nip1.Filter)
	OverwriteCountFilter      []func(ctx context.Context, filter *nip1.Filter)
	OverwriteRelayInformation []func(ctx context.Context, r *http.Request, info *nip11.RelayInfo) *nip11.RelayInfo
	StoreEvent                []func(ctx context.Context, event *nip1.Event) error
	DeleteEvent               []func(ctx context.Context, event *nip1.Event) error
	QueryEvents               []func(ctx context.Context, filter *nip1.Filter) (chan *nip1.Event, error)
	CountEvents               []func(ctx context.Context, filter *nip1.Filter) (int64, error)
	OnAuth                    []func(ctx context.Context, pubkey string)
	OnConnect                 []func(ctx context.Context)
	OnDisconnect              []func(ctx context.Context)
	OnEventSaved              []func(ctx context.Context, event *nip1.Event)

	// editing info will affect
	Info *nip11.RelayInfo

	// Default logger, as set by NewServer, is a stdlib logger prefixed with "[khatru-relay] ",
	// outputting to stderr.
	Log *log2.Logger

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
