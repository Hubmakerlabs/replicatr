package app

import (
	"net/http"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/config/base"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/acl"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relayinfo"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscriptionid"
	"github.com/Hubmakerlabs/replicatr/pkg/units"
	"github.com/gorilla/websocket"
	"github.com/puzpuzpuz/xsync/v2"
	"mleku.dev/git/atomic"
)

var Version = "v0.0.1"
var Software = "https://github.com/Hubmakerlabs/replicatr"

const (
	WriteWait           = 10 * time.Second
	PongWait            = 60 * time.Second
	PingPeriod          = 30 * time.Second
	ReadBufferSize      = 4096
	WriteBufferSize     = 4096
	MaxMessageSize  int = 128 * units.Kb
)

// function types used in the relay state
type (
	RejectEvent func(c context.T, ev *event.T) (rej bool,
		msg string)
	RejectFilter func(c context.T, id subscriptionid.T,
		f *filter.T) (reject bool, msg string)
	OverwriteFilter         func(c context.T, f *filter.T)
	OverrideDeletionOutcome func(c context.T, tgt, del *event.T) (ok bool,
		msg string)
	OverwriteResponseEvent    func(c context.T, ev *event.T)
	Events                    func(c context.T, ev *event.T) error
	Hook                      func(c context.T)
	OverwriteRelayInformation func(c context.T, r *http.Request,
		info *relayinfo.T) *relayinfo.T
	QueryEvents  func(c context.T, f *filter.T) (C event.C, err error)
	CountEvents  func(c context.T, f *filter.T) (cnt int, err error)
	OnEventSaved func(c context.T, ev *event.T)
)

type Relay struct {
	Ctx                    context.T
	WG                     *sync.WaitGroup
	Cancel                 context.F
	ServiceURL             atomic.String
	RejectEvent            []RejectEvent
	RejectFilter           []RejectFilter
	RejectCountFilter      []RejectFilter
	OverrideDeletion       []OverrideDeletionOutcome
	OverwriteResponseEvent []OverwriteResponseEvent
	OverwriteFilter        []OverwriteFilter
	OverwriteCountFilter   []OverwriteFilter
	OverwriteRelayInfo     []OverwriteRelayInformation
	StoreEvent             []Events
	DeleteEvent            []Events
	QueryEvents            []QueryEvents
	CountEvents            []CountEvents
	OnConnect              []Hook
	OnDisconnect           []Hook
	OnEventSaved           []OnEventSaved
	Config                 *base.Config
	Info                   *relayinfo.T
	// for establishing websockets
	upgrader websocket.Upgrader
	// keep a connection reference to all connected clients for Server.Shutdown
	clients *xsync.MapOf[*websocket.Conn, struct{}]
	// in case you call Server.Start
	Addr       string
	serveMux   *http.ServeMux
	httpServer *http.Server
	// websocket options
	// WriteWait is the time allowed to write a message to the peer.
	WriteWait time.Duration
	// PongWait is the time allowed to read the next pong message from the peer.
	PongWait time.Duration
	// PingPeriod is the tend pings to peer with this period. Must be less than
	// pongWait.
	PingPeriod     time.Duration
	MaxMessageSize int64    // Maximum message size allowed from peer.
	Whitelist      []string // whitelist of allowed IPs for access
	RelayPubHex    string
	RelayNpub      string
	// ACL is the list of users and privileges on this relay
	ACL *acl.T
}

// AuthCheck sends out a request if auth is required (this is an OnConnect
// method).
func (rl *Relay) AuthCheck(c context.T) {
	if rl.Info.Limitation.AuthRequired {
		// log.I.Ln("requesting auth")
		RequestAuth(c, "connection")
	}
}

func NewRelay(c context.T, cancel context.F,
	inf *relayinfo.T, conf *base.Config) (r *Relay) {

	var maxMessageLength = MaxMessageSize
	if inf.Limitation.MaxMessageLength > 0 {
		maxMessageLength = inf.Limitation.MaxMessageLength
	}
	pubKey, err := keys.GetPublicKey(conf.SecKey)
	chk.E(err)
	var npub string
	npub, err = bech32encoding.HexToNpub(pubKey)
	chk.E(err)
	inf.Software = Software
	inf.Version = Version
	inf.PubKey = pubKey
	r = &Relay{
		Ctx:    c,
		Cancel: cancel,
		Config: conf,
		Info:   inf,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  ReadBufferSize,
			WriteBufferSize: WriteBufferSize,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		clients: xsync.NewTypedMapOf[*websocket.Conn,
			struct{}](PointerHasher[websocket.Conn]),
		serveMux:       &http.ServeMux{},
		WriteWait:      WriteWait,
		PongWait:       PongWait,
		PingPeriod:     PingPeriod,
		MaxMessageSize: int64(maxMessageLength),
		Whitelist:      conf.Whitelist,
		RelayPubHex:    pubKey,
		RelayNpub:      npub,
		ACL:            &acl.T{},
	}
	log.I.F("relay chat pubkey: %s %s\n", pubKey, npub)
	// populate ACL with owners to start
	for _, owner := range r.Config.Owners {
		if err = r.ACL.AddEntry(&acl.Entry{
			Role:   acl.Owner,
			Pubkey: owner,
		}); chk.E(err) {
			continue
		}
		log.D.Ln("added owner pubkey", owner)
	}
	return
}
