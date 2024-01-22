package buffer

import "github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/text"

type Unmarshaler interface {
	Unmarshal(buf *text.Buffer) (e error)
}
