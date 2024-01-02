package kinds

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/array"
)

type T []kind.T

// ToArray converts to the generic array.T type ([]interface{})
func (ar T) ToArray() (a array.T) {
	a = make(array.T, len(ar))
	for i := range ar {
		a[i] = ar[i]
	}
	return
}

func FromIntSlice(is []int) (k T) {
	for i := range is {
		k = append(k, kind.T(is[i]))
	}
	return
}

// Clone makes a new kind.T with the same members.
func (ar T) Clone() (c T) {
	c = make(T, len(ar))
	for i := range ar {
		c[i] = ar[i]
	}
	return
}

// Contains returns true if the provided element is found in the kinds.T.
//
// Note that the request must use the typed kind.T or convert the number thus.
// Even if a custom number is found, this codebase does not have the logic to
// deal with the kind so such a search is pointless and for which reason static
// typing always wins. No mistakes possible with known quantities.
func (ar T) Contains(s kind.T) bool {
	for i := range ar {
		if ar[i] == s {
			return true
		}
	}
	return false
}

// Equals checks that the provided kind.T matches.
func (ar T) Equals(t1 T) bool {
	if len(ar) != len(t1) {
		return false
	}
	for i := range ar {
		if ar[i] != t1[i] {
			return false
		}
	}
	return true
}
