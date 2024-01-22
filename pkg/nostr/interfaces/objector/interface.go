package objector

import "github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/object"

// Objector is an interface for types that can be converted to object.T - a
// key-value type that can be summarised as
//
//	[]struct{Key string, Value interface{}}
//
// and allows non-set-like repetition and non-set-like canonical ordering.
//
// These are often called also "collection" in other languages but they are a
// frankenstein in implementation just like this here code demonstrates.
type Objector interface {
	ToObject() object.T
}
