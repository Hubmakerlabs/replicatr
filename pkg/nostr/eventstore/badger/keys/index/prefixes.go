package index

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/createdat"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/id"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/kinder"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/pubkey"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys/serial"
)

type P byte

// Key writes a key with the P prefix byte and an arbitrary list of
// keys.Element.
func (p P) Key(element ...keys.Element) (b []byte) {
	b = keys.Write(
		append([]keys.Element{New(byte(p))}, element...)...)
	// log.T.F("key %x", b)
	return
}

// B returns the index.P as a byte.
func (p P) B() byte { return byte(p) }

// I returns the index.P as an int (for use with the KeySizes.
func (p P) I() int { return int(p) }

// GetAsBytes todo wat is dis?
func GetAsBytes(prf ...P) (b [][]byte) {
	b = make([][]byte, len(prf))
	for i := range prf {
		b[i] = []byte{byte(prf[i])}
	}
	return
}

const (
	// Version is the key that stores the version number, the value is a 16-bit
	// integer (2 bytes)
	//
	//   [ 255 ][ 2 byte/16 bit version code ]
	Version P = 255
)
const (
	// Event is the prefix used with a Serial counter value provided by badgerDB to
	// provide conflict-free 8 byte 64-bit unique keys for event records, which
	// follows the prefix.
	//
	//   [ 0 ][ 8 bytes Serial ]
	Event P = iota

	// CreatedAt creates an index key that contains the unix
	// timestamp of the event record serial.
	//
	//   [ 1 ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	CreatedAt

	// Id contains the first 8 bytes of the ID of the event and the 8
	// byte Serial of the event record.
	//
	//   [ 2 ][ 8 bytes eventid.T prefix ][ 8 bytes Serial ]
	Id

	// Kind contains the kind and datestamp.
	//
	//   [ 3 ][ 2 bytes kind.T ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	Kind

	// Pubkey contains pubkey prefix and timestamp.
	//
	//   [ 4 ][ 8 bytes pubkey prefix ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	Pubkey

	// PubkeyKind contains pubkey prefix, kind and timestamp.
	//
	//   [ 5 ][ 8 bytes pubkey prefix ][ 2 bytes kind.T ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	PubkeyKind

	// Tag is for miscellaneous arbitrary length tags, with timestamp and event
	// serial after.
	//
	//   [ 6 ][ tag string 1 <= 100 bytes ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	Tag

	// Tag32 contains the 8 byte pubkey prefix, timestamp and serial.
	//
	//   [ 7 ][ 8 bytes pubkey prefix ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	Tag32

	// TagAddr contains the kind, pubkey prefix, value (index 2) of address tag (eg
	// relay address), followed by timestamp and serial.
	//
	//   [ 8 ][ 2 byte kind.T][ 8 byte pubkey prefix ][ network address ][ 8 byte timestamp.T ][ 8 byte Serial ]
	TagAddr

	// Counter is the eventid.T prefix, value stores the average time of access
	// (average of all access timestamps) and the size of the record.
	//
	//   [ 9 ][ 8 bytes Serial ] : value: [ 8 bytes timestamp ]
	Counter
)

// FilterPrefixes is a slice of the prefixes used by filter index to enable a loop
// for pulling events matching a serial
var FilterPrefixes = [][]byte{
	{CreatedAt.B()},
	{Id.B()},
	{Kind.B()},
	{Pubkey.B()},
	{PubkeyKind.B()},
	{Tag.B()},
	{Tag32.B()},
	{TagAddr.B()},
}

// KeySizes are the byte size of keys of each type of key prefix. int(P) or call the P.I() method
// corresponds to the index 1:1. For future index additions be sure to add the
// relevant KeySizes sum as it describes the data for a programmer.
var KeySizes = []int{
	// Event
	1 + serial.Len,
	// CreatedAt
	1 + createdat.Len + serial.Len,
	// Id
	1 + id.Len + serial.Len,
	// Kind
	1 + kinder.Len + createdat.Len + serial.Len,
	// Pubkey
	1 + pubkey.Len + createdat.Len + serial.Len,
	// PubkeyKind
	1 + pubkey.Len + kinder.Len + createdat.Len + serial.Len,
	// Tag (worst case scenario)
	1 + 100 + createdat.Len + serial.Len,
	// Tag32
	1 + pubkey.Len + createdat.Len + serial.Len,
	// TagAddr
	1 + kinder.Len + pubkey.Len + 100 + createdat.Len + serial.Len,
	// Counter
	1 + serial.Len,
}
