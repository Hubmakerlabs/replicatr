package index

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/eventstore/badger/keys"
)

type P byte

func (p P) Key(element ...keys.Element) (b []byte) {
	b = keys.Write(
		append([]keys.Element{New(byte(p))}, element...)...)
	// log.T.F("key %x", b)
	return
}

func (p P) Byte() byte { return byte(p) }

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

	// Event is the prefix used with a Serial counter value provided by badgerDB to
	// provide conflict-free 8 byte 64-bit unique keys for event records, which
	// follows the prefix.
	//
	//   [ 0 ][ 8 bytes Serial ]
	Event P = 0

	// CreatedAt creates an index key that contains the unix
	// timestamp of the event record serial.
	//
	//   [ 1 ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	CreatedAt P = 1

	// Id contains the first 8 bytes of the ID of the event and the 8
	// byte Serial of the event record.
	//
	//   [ 2 ][ 8 bytes eventid.T prefix ][ 8 bytes Serial ]
	Id P = 2

	// Kind contains the kind and datestamp.
	//
	//   [ 3 ][ 2 bytes kind.T ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	Kind P = 3

	// Pubkey contains pubkey prefix and timestamp.
	//
	//   [ 4 ][ 8 bytes pubkey prefix ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	Pubkey P = 4

	// PubkeyKind contains pubkey prefix, kind and timestamp.
	//
	//   [ 5 ][ 8 bytes pubkey prefix ][ 2 bytes kind.T ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	PubkeyKind P = 5

	// Tag is for miscellaneous arbitrary length tags, with timestamp and event
	// serial after.
	//
	//   [ 6 ][ tag string 1 <= 100 bytes ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	Tag P = 6

	// Tag32 contains the 8 byte pubkey prefix, timestamp and serial.
	//
	//   [ 7 ][ 8 bytes pubkey prefix ][ 8 bytes timestamp.T ][ 8 bytes Serial ]
	Tag32 P = 7

	// TagAddr contains the kind, pubkey prefix, value (index 2) of address tag (eg
	// relay address), followed by timestamp and serial.
	//
	//   [ 8 ][ 2 byte kind.T][ 8 byte pubkey prefix ][ network address ][ 8 byte timestamp.T ][ 8 byte Serial ]
	TagAddr P = 8

	// Counter is the eventid.T prefix, value stores the average time of access
	// (average of all access timestamps) and the size of the record.
	//
	// Size is set to 0 if record data has been pruned as a flag marking a pruned
	// record.
	//
	//   [ 9 ][ 16 byte eventid.T prefix ][ 8 bytes Serial ]
	Counter P = 9
)
