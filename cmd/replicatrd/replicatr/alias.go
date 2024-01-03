package replicatr

import (
	"context"
	"net/http"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip1"
	"github.com/fasthttp/websocket"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
	"github.com/puzpuzpuz/xsync/v2"
)

// aliases so we can swap out to another package with only here changed
type (
	Ctx             = context.Context
	Info            = nip11.RelayInformationDocument
	Event           = nostr.Event
	Filter          = nostr.Filter
	Filters         = nostr.Filters
	TagMap          = nostr.TagMap
	EventEnvelope   = nostr.EventEnvelope
	OKEnvelope      = nostr.OKEnvelope
	EventID         = nip1.EventID
	CountEnvelope   = nostr.CountEnvelope
	ClosedEnvelope  = nostr.ClosedEnvelope
	ReqEnvelope     = nostr.ReqEnvelope
	EOSEEnvelope    = nostr.EOSEEnvelope
	CloseEnvelope   = nostr.CloseEnvelope
	AuthEnvelope    = nostr.AuthEnvelope
	NoticeEnvelope  = nostr.NoticeEnvelope
	Conn            = websocket.Conn
	Request         = http.Request
	ResponseWriter  = http.ResponseWriter
	Mutex           = sync.Mutex
	WaitGroup       = sync.WaitGroup
	CancelCauseFunc = context.CancelCauseFunc
	ListenerMap     = *xsync.MapOf[string, *Listener]
	Timestamp       = nostr.Timestamp
)
