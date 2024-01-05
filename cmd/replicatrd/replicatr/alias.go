package replicatr

import (
	"github.com/puzpuzpuz/xsync/v2"
)

type (
	ListenerMap = *xsync.MapOf[string, *Listener]
)
