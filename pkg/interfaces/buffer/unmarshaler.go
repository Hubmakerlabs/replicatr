package buffer

import "github.com/Hubmakerlabs/replicatr/pkg/wire/text"

type Unmarshaler interface {
	Unmarshal(buf *text.Buffer) (e error)
}
