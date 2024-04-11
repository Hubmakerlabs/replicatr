package badger

const (
	// Megabyte is the standard 1 million byte units.
	Megabyte = 1000 * 1000

	// SerialLen is the length of serial values used for conflict resistant
	// keys.
	SerialLen = 8

	// TimestampLen is the standard 64 bit, 8 byte unix timestamp
	TimestampLen = 8

	// IDPrefixLen is the length of the prefix trimmed out of a 256 bit SHA256
	// hash that is the "id" field of an event.
	IDPrefixLen = 8

	// PubkeyPrefixLen is the length of the prefix trimmed out of a public key
	// field of an event used in index keys.
	PubkeyPrefixLen = 8

	// KindLen is the length of bytes to store a kind.T - 2 bytes/16 bits
	KindLen = 2

	// PrefixLen is the length of the database key prefixes for each type of
	// record. It is 1, but this makes it unambiguous what it means.
	PrefixLen = 1
)
