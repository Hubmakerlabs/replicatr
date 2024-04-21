// Package l2 is a testing data store that implements a level 2 cache for events
// with a badger eventstore.
//
// This is a testing environment for building cache strategies.
package l2

import (
	"os"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/del"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

// GetDefaultBackend returns a reasonably configured badger.Backend.
//
// The variadic params correspond to DBSizeLimit, DBLowWater, DBHighWater and
// GCFrequency as an integer multiplier of number of seconds.
//
// Note that the cancel function for the context needs to be managed by the
// caller.
func GetDefaultBackend(
	Ctx context.T,
	WG *sync.WaitGroup,
	path string,
	params ...int,
) (b *badger.Backend) {
	var mb, lw, hw, freq = 0, 86, 92, 60
	switch len(params) {
	case 4:
		freq = params[3]
		fallthrough
	case 3:
		hw = params[2]
		fallthrough
	case 2:
		lw = params[1]
		fallthrough
	case 1:
		mb = params[0]
	}
	b = &badger.Backend{
		Ctx:         Ctx,
		WG:          WG,
		Path:        path,
		MaxLimit:    badger.DefaultMaxLimit,
		DBSizeLimit: mb * badger.Megabyte,
		DBLowWater:  lw,
		DBHighWater: hw,
		GCFrequency: time.Duration(freq) * time.Second,
		PruneFunc:   Prune(),
		GCCountFunc: GCCount(),
	}
	return
}

func GCCount() func(ifc any) (deleteItems del.Items, err error) {
	return func(ifc any) (deleteItems del.Items, err error) {

		return
	}
}

func Prune() func(ifc any, deleteItems del.Items) (err error) {
	return func(ifc any, deleteItems del.Items) (err error) {

		return
	}
}
