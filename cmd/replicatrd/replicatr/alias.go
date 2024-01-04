package replicatr

import (
	"context"
	"net/http"
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/OK"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/auth"
	close2 "github.com/Hubmakerlabs/replicatr/pkg/go-nostr/close"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/closed"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/count"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/eose"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip11"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/req"
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
	Filter          = filter.Filter
	Filters         = filter.Filters
	TagMap          = filter.TagMap
	OKEnvelope      = OK.OKEnvelope
	EventID         = eventid.EventID
	CountEnvelope   = count.CountEnvelope
	ClosedEnvelope  = closed.ClosedEnvelope
	ReqEnvelope     = req.ReqEnvelope
	EOSEEnvelope    = eose.EOSEEnvelope
	CloseEnvelope   = close2.CloseEnvelope
	AuthEnvelope    = auth.AuthEnvelope
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
