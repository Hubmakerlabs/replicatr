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

const (
	dbVersionKey          byte = 255
	rawEventStorePrefix   byte = 0
	indexCreatedAtPrefix  byte = 1
	indexIdPrefix         byte = 2
	indexKindPrefix       byte = 3
	indexPubkeyPrefix     byte = 4
	indexPubkeyKindPrefix byte = 5
	indexTagPrefix        byte = 6
	indexTag32Prefix      byte = 7
	indexTagAddrPrefix    byte = 8
)

var _ eventstore.Store = (*Backend)(nil)

type Backend struct {
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

	*badger.DB
	seq *badger.Sequence

	Ctx context.T
	WG  *sync.WaitGroup
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
		b.MaxLimit = 500
	}

	return nil
}

func (b *Backend) Close() {
	b.DB.Close()
	b.seq.Release()
}

// SerialKey gets a sequence number from the badger DB backend, prefixed with a key
// type code 0 Event.
//
// This value is used as the key for a raw event record.
func (b *Backend) SerialKey() (idx []byte, ser []byte) {
	// v := b.Serial()
	// vb := make([]byte, 9)
	// vb[0]
	// binary.BigEndian.PutUint64(vb[1:], v)
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

func (b *Backend) SerialBytes() (ser []byte, err error) {
	var serU64 uint64
	if serU64, err = b.Serial(); chk.E(err) {
		panic(err)
	}
	ser = make([]byte, serial.Len)
	binary.BigEndian.PutUint64(ser, serU64)
	return
}
