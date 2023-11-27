package nostr

import (
	"mleku.online/git/replicatr/pkg/jsontext"
	"strings"
)

// The tag position meanings so they are clear when reading.
const (
	TagKey = iota
	TagValue
	TagRelay
)

// Tag is a list of strings with a literal ordering.
//
// Not a set, there can be repeating elements.
type Tag []string

// StartsWith checks a tag has the same initial set of elements.
//
// The last element is treated specially in that it is considered to match if
// the candidate has the same initial substring as its corresponding element.
func (tag Tag) StartsWith(prefix []string) bool {
	prefixLen := len(prefix)

	if prefixLen > len(tag) {
		return false
	}
	// check initial elements for equality
	for i := 0; i < prefixLen-1; i++ {
		if prefix[i] != tag[i] {
			return false
		}
	}
	// check last element just for a prefix
	return strings.HasPrefix(tag[prefixLen-1], prefix[prefixLen-1])
}

// Key returns the first element of the tags.
func (tag Tag) Key() string {
	if len(tag) > TagKey {
		return tag[TagKey]
	}
	return ""
}

// Value returns the second element of the tag.
func (tag Tag) Value() string {
	if len(tag) > TagValue {
		return tag[TagValue]
	}
	return ""
}

// Relay returns the third element of the tag.
func (tag Tag) Relay() string {
	if (tag.Key() == "e" || tag.Key() == "p") && len(tag) > TagRelay {
		return NormalizeURL(tag[TagRelay])
	}
	return ""
}

// Marshal Tag. Used for Serialization so string escaping should be as in
// RFC8259.
func (tag Tag) marshalTo(dst []byte) []byte {
	dst = append(dst, '[')
	for i, s := range tag {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, jsontext.EscapeJSONStringAndWrap(s)...)
	}
	dst = append(dst, ']')
	return dst
}
