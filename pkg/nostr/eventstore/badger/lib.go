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
	"github.com/Hubmakerlabs/replicatr/pkg/units"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
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
	GCFrequency    time.Duration
	HasL2          bool
	BlockCacheSize int
	// DB is the badger db interface
	*badger.DB
	// seq is the monotonic collision free index for raw event storage.
	seq *badger.Sequence
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
	blockCacheSize int,
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
		Ctx:            Ctx,
		WG:             WG,
		Path:           path,
		MaxLimit:       DefaultMaxLimit,
		DBSizeLimit:    sizeLimit,
		DBLowWater:     lw,
		DBHighWater:    hw,
		GCFrequency:    time.Duration(freq) * time.Second,
		HasL2:          hasL2,
		BlockCacheSize: blockCacheSize,
	}
	return
}

func (b *Backend) Init() (err error) {
	log.I.Ln("opening badger event store at", b.Path)
	opts := badger.DefaultOptions(b.Path)
	opts.Compression = options.None
	opts.BlockCacheSize = int64(b.BlockCacheSize)
	opts.BlockSize = units.Mb
	opts.CompactL0OnClose = true
	opts.LmaxCompaction = true
	opts.Compression = options.ZSTD
	// opts.Logger = logger{0, b.Path}
	if b.DB, err = badger.Open(opts); chk.E(err) {
		return err
	}
	log.I.Ln("getting event store sequence index", b.Path)
	if b.seq, err = b.DB.GetSequence([]byte("events"), 1000); chk.E(err) {
		return err
	}
	log.I.Ln("running migrations", b.Path)
	if err = b.runMigrations(); chk.E(err) {
		return log.E.Err("error running migrations: %w; %s", err, b.Path)
	}
	if b.MaxLimit == 0 {
		b.MaxLimit = DefaultMaxLimit
	}
	if b.DBSizeLimit > 0 {
		go b.GarbageCollector()
	} else {
		go b.GCCount()
		// go b.IndexGCCount()
	}
	return nil
}

func (b *Backend) Close() { _, _ = b.DB.Close(), b.seq.Release() }

// SerialKey returns a key used for storing events, and the raw serial counter
// bytes to copy into index keys.
func (b *Backend) SerialKey() (idx []byte, ser *serial.T) {
	var err error
	var s []byte
	if s, err = b.SerialBytes(); chk.E(err) {
		panic(err)
	}
	ser = serial.New(s)
	return index.Event.Key(ser), ser
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
	err = b.DB.Update(fn)
	return
}

func (b *Backend) View(fn func(txn *badger.Txn) (err error)) (err error) {
	err = b.DB.View(fn)
	return
}
