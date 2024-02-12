package badger

import (
	"encoding/binary"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip11"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"mleku.online/git/slog"
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
	Path     string
	MaxLimit int
	*badger.DB
	seq *badger.Sequence
}

func (b *Backend) Init(inf *nip11.Info) (err error) {
	badgerOpts := badger.
		DefaultOptions(b.Path).
		WithCompactL0OnClose(true).
		WithCompression(options.ZSTD)
	ll := slog.GetLogLevel()
	switch ll {
	case slog.Off, slog.Fatal, slog.Error:
		badgerOpts = badgerOpts.WithLoggingLevel(badger.ERROR)
	case slog.Info:
		badgerOpts = badgerOpts.WithLoggingLevel(badger.INFO)
	case slog.Warn:
		badgerOpts = badgerOpts.WithLoggingLevel(badger.WARNING)
	case slog.Debug, slog.Trace:
		badgerOpts = badgerOpts.WithLoggingLevel(badger.DEBUG)
	}
	var db *badger.DB
	db, err = badger.Open(badgerOpts)
	if err != nil {
		return err
	}
	b.DB = db
	if b.seq, err = db.GetSequence([]byte("events"), 1000); chk.E(err) {
		return err
	}
	if err = b.runMigrations(); chk.E(err) {
		return fmt.Errorf("error running migrations: %w", err)
	}
	b.MaxLimit = inf.Limitation.MaxLimit
	return nil
}

func (b *Backend) Close() {
	log.E.Chk(b.DB.Close())
	log.E.Chk(b.seq.Release())
}

func (b *Backend) Serial() []byte {
	v, _ := b.seq.Next()
	vb := make([]byte, 5)
	vb[0] = rawEventStorePrefix
	binary.BigEndian.PutUint32(vb[1:], uint32(v))
	return vb
}
