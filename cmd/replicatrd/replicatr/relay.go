package replicatr

import (
	"net/http"
	"os"
	"time"

	log2 "github.com/Hubmakerlabs/replicatr/pkg/log"

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
	RejectEvent               func(ctx Ctx, event *Event) (reject bool, msg string)
	RejectFilter              func(ctx Ctx, filter *Filter) (reject bool, msg string)
	OverwriteFilter           func(ctx Ctx, filter *Filter)
	OverwriteDeletionOutcome  func(ctx Ctx, target *Event, del *Event) (accept bool, msg string)
	OverwriteResponseEvent    func(ctx Ctx, event *Event)
	Events                    func(ctx Ctx, event *Event) error
	Hook                      func(ctx Ctx)
	OverwriteRelayInformation func(ctx Ctx, r *Request, info *Info) *Info
	QueryEvents               func(ctx Ctx, filter *Filter) (eventC chan *Event, e error)
	CountEvents               func(ctx Ctx, filter *Filter) (c int64, e error)
	OnEventSaved              func(ctx Ctx, event *Event)
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
	Info *Info
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

func NewRelay(appName string) (r *Relay) {
	r = &Relay{
		Log: log2.New(os.Stderr, appName, 0),
		Info: &Info{
			Software:      "https://github.com/Hubmakerlabs/replicatr/cmd/khatru",
			Version:       "n/a",
			SupportedNIPs: make([]int, 0),
		},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  ReadBufferSize,
			WriteBufferSize: WriteBufferSize,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		clients:        xsync.NewTypedMapOf[*websocket.Conn, struct{}](pointerHasher[websocket.Conn]),
		serveMux:       &http.ServeMux{},
		WriteWait:      WriteWait,
		PongWait:       PongWait,
		PingPeriod:     PingPeriod,
		MaxMessageSize: MaxMessageSize,
	}
	return
}
