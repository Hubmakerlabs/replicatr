package kinds

import (
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/wire/array"
)

type T []kind.T

// ToArray converts to the generic array.T type ([]interface{})
func (k T) ToArray() (a array.T) {
	a = make(array.T, len(k))
	for i := range k {
		a[i] = k[i]
	}
	return
}

func FromIntSlice(is []int) (k T) {
	for i := range is {
		k = append(k, kind.T(is[i]))
	}
	return
}

func (k T) ToUint16() (o []uint16) {
	for i := range k {
		o = append(o, uint16(k[i]))
	}
	return
}

// Clone makes a new kind.T with the same members.
func (k T) Clone() (c T) {
	c = make(T, len(k))
	for i := range k {
		c[i] = k[i]
	}
	return
}

// Contains returns true if the provided element is found in the kinds.T.
//
// Note that the request must use the typed kind.T or convert the number thus.
// Even if a custom number is found, this codebase does not have the logic to
// deal with the kind so such a search is pointless and for which reason static
// typing always wins. No mistakes possible with known quantities.
func (k T) Contains(s kind.T) bool {
	for i := range k {
		if k[i] == s {
			return true
		}
	}
	return false
}

// Equals checks that the provided kind.T matches.
func (k T) Equals(t1 T) bool {
	if len(k) != len(t1) {
		return false
	}
	for i := range k {
		if k[i] != t1[i] {
			return false
		}
	}
	return true
}
