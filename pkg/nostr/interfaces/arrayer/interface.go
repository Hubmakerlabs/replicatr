package arrayer

import "github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"

// I is an interface for a type that can return an array.T - or in other
// words []interface{} made into concrete.
type I interface {
	ToArray() array.T
}
