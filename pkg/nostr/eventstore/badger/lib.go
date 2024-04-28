package badger

import (
	"encoding/binary"
	"os"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/del"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/dgraph-io/badger/v4"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

var _ eventstore.Store = (*Backend)(nil)

type PruneFunc func(ifc any, deleteItems del.Items) (err error)

type GCCountFunc func(ifc any) (deleteItems del.Items, err error)

type Backend struct {
	Ctx      context.T
	WG       *sync.WaitGroup
	Path     string
	LogLevel int
	// MaxLimit is the largest a single event JSON can be, in bytes.
	MaxLimit int
	// DBSizeLimit is the number of bytes we want to keep the data store from
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
	HasL2       bool
	// DB is the badger db interface
	*badger.DB
	// seq is the monotonic collision free index for raw event storage.
	seq *badger.Sequence
	// bMx is a lock that prevents more than one operation running at a time
	bMx sync.RWMutex
}

const DefaultMaxLimit = 1024

// GetBackend returns a reasonably configured badger.Backend.
//
// The variadic params correspond to DBSizeLimit, DBLowWater, DBHighWater and
// GCFrequency as an integer multiplier of number of seconds.
//
// Note that the cancel function for the context needs to be managed by the
// caller.
func GetBackend(
	Ctx context.T,
	WG *sync.WaitGroup,
	path string,
	hasL2 bool,
	logLevel int,
	params ...int,
) (b *Backend) {
	var sizeLimit, lw, hw, freq = 0, 86, 92, 60
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
		sizeLimit = params[0]
	}
	b = &Backend{
		Ctx:         Ctx,
		WG:          WG,
		Path:        path,
		LogLevel:    logLevel,
		MaxLimit:    DefaultMaxLimit,
		DBSizeLimit: sizeLimit,
		DBLowWater:  lw,
		DBHighWater: hw,
		GCFrequency: time.Duration(freq) * time.Second,
		HasL2:       hasL2,
	}
	return
}

func (b *Backend) Init() (err error) {
	if b.DB, err = badger.Open(badger.DefaultOptions(b.Path).WithLogger(logger(b.LogLevel))); chk.E(err) {
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
	if b.DBSizeLimit > 0 {
		go b.GarbageCollector()
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
	// log.T.F("serial %x", ser)
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

func (b *Backend) Update(fn func(txn *badger.Txn) (err error)) (err error) {
	b.bMx.Lock()
	err = b.DB.Update(fn)
	b.bMx.Unlock()
	return
}

func (b *Backend) View(fn func(txn *badger.Txn) (err error)) (err error) {
	b.bMx.RLock()
	err = b.DB.View(fn)
	b.bMx.RUnlock()
	return
}
