package badger

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/index"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
	"github.com/dgraph-io/badger/v4"
)

var _ eventstore.Store = (*Backend)(nil)

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
	Delete func(serials DeleteItems) (err error)

	*badger.DB
	seq *badger.Sequence
}

const DefaultMaxLimit = 1024

// GetDefaultBackend returns a reasonably configured badger.Backend.
func GetDefaultBackend(
	Ctx context.T,
	WG *sync.WaitGroup,
	path string,
	mb int, // event size limit in Mb, 0 to disable.
) (b *Backend) {
	b = &Backend{
		Ctx:         Ctx,
		WG:          WG,
		Path:        path,
		MaxLimit:    DefaultMaxLimit,
		DBSizeLimit: mb * Megabyte,
		DBLowWater:  86,
		DBHighWater: 92,
		GCFrequency: 5 * time.Second,
	}
	b.Delete = b.BadgerDelete
	return
}

func (b *Backend) Init() error {
	db, err := badger.Open(badger.DefaultOptions(b.Path))
	if err != nil {
		return err
	}
	b.DB = db
	b.seq, err = db.GetSequence([]byte("events"), 1000)
	if err != nil {
		return err
	}
	if err := b.runMigrations(); err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}
	if b.MaxLimit == 0 {
		b.MaxLimit = DefaultMaxLimit
	}
	// set Delete function if it's empty
	if b.Delete == nil {
		b.Delete = b.BadgerDelete
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
