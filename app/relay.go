package app

import (
	"net/http"
	"time"

	"github.com/Hubmakerlabs/replicatr/app/acl"
	"github.com/fasthttp/websocket"
	"github.com/puzpuzpuz/xsync/v2"
	"mleku.dev/git/atomic"
	"mleku.dev/git/nostr/bech32encoding"
	"mleku.dev/git/nostr/context"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/keys"
	"mleku.dev/git/nostr/relayinfo"
	"mleku.dev/git/nostr/subscriptionid"
)

var Version = "v0.0.1"
var Software = "https://github.com/Hubmakerlabs/replicatr"

const (
	WriteWait           = 10 * time.Second
	PongWait            = 60 * time.Second
	PingPeriod          = 30 * time.Second
	ReadBufferSize      = 4096
	WriteBufferSize     = 4096
	MaxMessageSize  int = 512000 // ???
)

// function types used in the relay state
type (
	RejectEvent               func(c context.T, ev *event.T) (rej bool, msg string)
	RejectFilter              func(c context.T, id subscriptionid.T, f *filter.T) (reject bool, msg string)
	OverwriteFilter           func(c context.T, f *filter.T)
	OverrideDeletionOutcome   func(c context.T, tgt, del *event.T) (ok bool, msg string)
	OverwriteResponseEvent    func(c context.T, ev *event.T)
	Events                    func(c context.T, ev *event.T) error
	Hook                      func(c context.T)
	OverwriteRelayInformation func(c context.T, r *http.Request, info *relayinfo.T) *relayinfo.T
	QueryEvents               func(c context.T, C chan *event.T, f *filter.T) (err error)
	CountEvents               func(c context.T, f *filter.T) (cnt int64, err error)
	OnEventSaved              func(c context.T, ev *event.T)
)

type Relay struct {
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
	Config                 *Config
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

func (rl *Relay) AuthCheck(c context.T) {
	if rl.Info.Limitation.AuthRequired {
		RequestAuth(c)
	}
}

func NewRelay(inf *relayinfo.T,
	conf *Config) (r *Relay) {

	var maxMessageLength = MaxMessageSize
	if inf.Limitation.MaxMessageLength > 0 {
		maxMessageLength = inf.Limitation.MaxMessageLength
	}
	pubKey, err := keys.GetPublicKey(conf.SecKey)
	chk.E(err)
	var npub string
	npub, err = bech32encoding.EncodePublicKey(pubKey)
	chk.E(err)
	r = &Relay{
		Config: conf,
		Info:   relayinfo.NewInfo(inf),
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
	log.I.F("relay chat pubkey: %s %s", pubKey, npub)
	r.Info.Software = Software
	r.Info.Version = Version
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
