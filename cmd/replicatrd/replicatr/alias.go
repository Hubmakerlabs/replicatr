package replicatr

import (
	"context"
	"net/http"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip11"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventid"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/notice"
	"github.com/fasthttp/websocket"
	"github.com/puzpuzpuz/xsync/v2"
)

// aliases so we can swap out to another package with only here changed
type (
	Ctx             = context.Context
	Info            = nip11.RelayInformationDocument
	Event           = event.T
	Filter          = nostr.Filter
	Filters         = nostr.Filters
	TagMap          = nostr.TagMap
	OKEnvelope      = nostr.OKEnvelope
	EventID         = eventid.EventID
	CountEnvelope   = nostr.CountEnvelope
	ClosedEnvelope  = nostr.ClosedEnvelope
	ReqEnvelope     = nostr.ReqEnvelope
	EOSEEnvelope    = nostr.EOSEEnvelope
	CloseEnvelope   = nostr.CloseEnvelope
	AuthEnvelope    = nostr.AuthEnvelope
	NoticeEnvelope  = notice.Envelope
	Conn            = websocket.Conn
	Request         = http.Request
	ResponseWriter  = http.ResponseWriter
	Mutex           = sync.Mutex
	WaitGroup       = sync.WaitGroup
	CancelCauseFunc = context.CancelCauseFunc
	ListenerMap     = *xsync.MapOf[string, *Listener]
	Timestamp       = timestamp.Timestamp
)
