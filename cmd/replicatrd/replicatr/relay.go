package replicatr

import (
	"net/http"
	"os"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip11"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
	"github.com/fasthttp/websocket"
	"github.com/puzpuzpuz/xsync/v2"
)

const (
	WriteWait             = 10 * time.Second
	PongWait              = 60 * time.Second
	PingPeriod            = 30 * time.Second
	ReadBufferSize        = 4096
	WriteBufferSize       = 4096
	MaxMessageSize  int64 = 512000 // ???
)

// function types used in the relay state
type (
	RejectEvent               func(c context.T, event *event.T) (reject bool, msg string)
	RejectFilter              func(c context.T, f *filter.T) (reject bool, msg string)
	OverwriteFilter           func(c context.T, f *filter.T)
	OverwriteDeletionOutcome  func(c context.T, target, del *event.T) (accept bool, msg string)
	OverwriteResponseEvent    func(c context.T, ev *event.T)
	Events                    func(c context.T, ev *event.T) error
	Hook                      func(c context.T)
	OverwriteRelayInformation func(c context.T, r *http.Request, info *nip11.Info) *nip11.Info
	QueryEvents               func(c context.T, f *filter.T) (eventC chan *event.T, e error)
	CountEvents               func(c context.T, f *filter.T) (cnt int64, e error)
	OnEventSaved              func(c context.T, ev *event.T)
)

type Relay struct {
	ServiceURL               string
	RejectEvent              []RejectEvent
	RejectFilter             []RejectFilter
	RejectCountFilter        []RejectFilter
	OverwriteDeletionOutcome []OverwriteDeletionOutcome
	OverwriteResponseEvent   []OverwriteResponseEvent
	OverwriteFilter          []OverwriteFilter
	OverwriteCountFilter     []OverwriteFilter
	OverwriteRelayInfo       []OverwriteRelayInformation
	StoreEvent               []Events
	DeleteEvent              []Events
	QueryEvents              []QueryEvents
	CountEvents              []CountEvents
	OnConnect                []Hook
	OnDisconnect             []Hook
	OnEventSaved             []OnEventSaved
	// editing info will affect
	Info *nip11.Info
	*slog.Log
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

func NewRelay(appName string) (r *Relay) {
	r = &Relay{
		Log: slog.New(os.Stderr, appName),
		Info: &nip11.Info{
			Software:      "https://github.com/Hubmakerlabs/replicatr/cmd/replicatrd",
			Version:       "n/a",
			SupportedNIPs: make([]int, 0),
		},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  ReadBufferSize,
			WriteBufferSize: WriteBufferSize,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		clients:        xsync.NewTypedMapOf[*websocket.Conn, struct{}](PointerHasher[websocket.Conn]),
		serveMux:       &http.ServeMux{},
		WriteWait:      WriteWait,
		PongWait:       PongWait,
		PingPeriod:     PingPeriod,
		MaxMessageSize: MaxMessageSize,
	}
	return
}
