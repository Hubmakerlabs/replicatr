package badger

import (
	"encoding/binary"
	"os"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/del"
	"github.com/dgraph-io/badger/v4"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

var _ eventstore.Store = (*Backend)(nil)

type PruneFunc func(ifc any, deleteItems del.Items) (err error)

type GCCountFunc func(ifc any) (deleteItems del.Items, err error)

type Backend struct {
	Ctx  context.T
	WG   *sync.WaitGroup
	Path string
	// MaxLimit is the largest a single event JSON can be, in bytes.
	MaxLimit int
	// DBSizeLimit is the number of Mb we want to keep the data store from
	// exceeding.
	DBSizeLimit int
	// DBLowWater is the percentage of DBSizeLimit a GC run will reduce the used
	// storage down to.
	DBLowWater int
	// DBHighWater is the trigger point at which a GC run should start if
	// exceeded.
	DBHighWater int
	// GCFrequency is the frequency of checks of the current utilisation.
	GCFrequency time.Duration
	// Delete is a closure that implements the garbage collection prune operation.
	//
	// This is to enable multi-level caching as well as maintaining a limit of
	// storage usage.
	PruneFunc func(ifc any, deleteItems del.Items) (err error)
	// GCCount is a function that iterates the access timestamp records to determine
	// the most stale events and return the list of serials the Delete function
	// should operate on.
	//
	// Note that this function needs to be able to access the DBSizeLimit,
	// DBLowWater and DBHighWater values as this is the configuration for garbage
	// collection.
	GCCountFunc func(ifc any) (deleteItems del.Items, err error)
	// L2 is a secondary event store, that, if used, should be loaded in combination
	// with Delete and GCCount methods to enable second level database storage
	// functionality.
	L2 eventstore.Store
	// DB is the badger db interface
	*badger.DB
	// seq is the monotonic collision free index for raw event storage.
	seq *badger.Sequence
}

// GCCount calls the closure that was loaded for the back end second level cache.
func (b *Backend) GCCount() (deleteItems del.Items, err error) { return b.GCCountFunc(b) }

// Prune calls the closure that was loaded for the back end second level cache.
func (b *Backend) Prune(deleteItems del.Items) (err error) { return b.PruneFunc(b, deleteItems) }

const DefaultMaxLimit = 1024

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
) (b *Backend) {
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
	b = &Backend{
		Ctx:         Ctx,
		WG:          WG,
		Path:        path,
		MaxLimit:    DefaultMaxLimit,
		DBSizeLimit: mb * Megabyte,
		DBLowWater:  lw,
		DBHighWater: hw,
		GCFrequency: time.Duration(freq) * time.Second,
		PruneFunc:   Prune(),
		GCCountFunc: GCCount(),
	}
	return
}

func (b *Backend) Init() (err error) {
	if b.DB, err = badger.Open(badger.DefaultOptions(b.Path)); chk.E(err) {
		return err
	}
	if b.seq, err = b.DB.GetSequence([]byte("events"), 1000); chk.E(err) {
		return err
	}
	if err = b.runMigrations(); chk.E(err) {
		return log.E.Err("error running migrations: %w", err)
	}
	if b.MaxLimit == 0 {
		b.MaxLimit = DefaultMaxLimit
	}
	return nil
}

func (b *Backend) Close() { _, _ = b.DB.Close(), b.seq.Release() }

// SerialKey returns a key used for storing events, and the raw serial counter
// bytes to copy into index keys.
func (b *Backend) SerialKey() (idx []byte, ser []byte) {
	var err error
	if ser, err = b.SerialBytes(); chk.E(err) {
		panic(err)
	}
	return index.Event.Key(serial.New(ser)), ser
}

func (b *Backend) Serial() (ser uint64, err error) {
	if ser, err = b.seq.Next(); chk.E(err) {
	}
	log.T.F("serial %x", ser)
	return
}

// SerialBytes returns a new serial value, used to store an event record with a
// conflict-free unique code (it is a monotonic, atomic, ascending counter).
func (b *Backend) SerialBytes() (ser []byte, err error) {
	var serU64 uint64
	if serU64, err = b.Serial(); chk.E(err) {
		panic(err)
	}
	ser = make([]byte, serial.Len)
	binary.BigEndian.PutUint64(ser, serU64)
	return
}
