package buffer

import "mleku.dev/git/nostr/wire/text"

type Unmarshaler interface {
	Unmarshal(buf *text.Buffer) (err error)
}
