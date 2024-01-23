package badger

import (
	"encoding/binary"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/slog"
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

var _ eventstore.Store = (*BadgerBackend)(nil)

type BadgerBackend struct {
	Path     string
	MaxLimit int
	*slog.Log
	*badger.DB
	seq *badger.Sequence
}

func (b *BadgerBackend) Init() (e error) {
	db, e := badger.Open(badger.DefaultOptions(b.Path))
	if e != nil {
		return e
	}
	b.DB = db
	b.seq, e = db.GetSequence([]byte("events"), 1000)
	if e != nil {
		return e
	}

	if e := b.runMigrations(); e != nil {
		return fmt.Errorf("error running migrations: %w", e)
	}

	if b.MaxLimit == 0 {
		b.MaxLimit = 500
	}

	return nil
}

func (b *BadgerBackend) Close() {
	log.E.Chk(b.DB.Close())
	log.E.Chk(b.seq.Release())
}

func (b *BadgerBackend) Serial() []byte {
	v, _ := b.seq.Next()
	vb := make([]byte, 5)
	vb[0] = rawEventStorePrefix
	binary.BigEndian.PutUint32(vb[1:], uint32(v))
	return vb
}
