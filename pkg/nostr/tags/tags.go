package tags

import (
	"encoding/json"
	"errors"
	t "mleku.online/git/replicatr/pkg/nostr/tag"
)

// T is a list of T - which are lists of string elements with ordering and
// no uniqueness constraint (not a set).
type T []t.T

// GetFirst gets the first t in tags that matches the prefix, see [T.StartsWith]
func (t T) GetFirst(tagPrefix []string) *t.T {
	for _, v := range t {
		if v.StartsWith(tagPrefix) {
			return &v
		}
	}
	return nil
}

// GetLast gets the last t in tags that matches the prefix, see [T.StartsWith]
func (t T) GetLast(tagPrefix []string) *t.T {
	for i := len(t) - 1; i >= 0; i-- {
		v := t[i]
		if v.StartsWith(tagPrefix) {
			return &v
		}
	}
	return nil
}

// GetAll gets all the tags that match the prefix, see [T.StartsWith]
func (t T) GetAll(tagPrefix []string) T {
	result := make(T, 0, len(t))
	for _, v := range t {
		if v.StartsWith(tagPrefix) {
			result = append(result, v)
		}
	}
	return result
}

// FilterOut removes all tags that match the prefix, see [T.StartsWith]
func (t T) FilterOut(tagPrefix []string) T {
	filtered := make(T, 0, len(t))
	for _, v := range t {
		if !v.StartsWith(tagPrefix) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// AppendUnique appends a t if it doesn't exist yet, otherwise does nothing.
// the uniqueness comparison is done based only on the first 2 elements of the t.
func (t T) AppendUnique(tag t.T) T {
	n := len(tag)
	if n > 2 {
		n = 2
	}
	if t.GetFirst(tag[:n]) == nil {
		return append(t, tag)
	}
	return t
}

// Scan parses a string or raw bytes that should be a string and embeds the
// values into the tags variable from which this method is invoked.
func (t T) Scan(src any) (err error) {
	var jtags []byte
	switch v := src.(type) {
	case []byte:
		jtags = v
	case string:
		jtags = []byte(v)
	default:
		return errors.New("couldn't scan t, it's not a json string")
	}
	err = json.Unmarshal(jtags, &t)
	return
}

// ContainsAny returns true if any of the strings given in `values` matches any
// of the t elements.
func (t T) ContainsAny(tagName string, values []string) bool {
	for _, v := range t {
		if len(v) < 2 {
			continue
		}
		if v.Key() != tagName {
			continue
		}
		for _, candidate := range values {
			if v.Value() == candidate {
				return true
			}
		}
	}
	return false
}

// MarshalTo appends the JSON encoded byte of T as [][]string to dst. String
// escaping is as described in RFC8259.
func (t T) MarshalTo(dst []byte) []byte {
	dst = append(dst, '[')
	for i, t := range t {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = t.MarshalTo(dst)
	}
	dst = append(dst, ']')
	return dst
}
