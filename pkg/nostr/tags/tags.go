package tags

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

// T is a list of T - which are lists of string elements with ordering and no
// uniqueness constraint (not a set).
type T []tag.T

// GetFirst gets the first tag in tags that matches the prefix, see
// [T.StartsWith]
func (t T) GetFirst(tagPrefix []string) *tag.T {
	for _, v := range t {
		if v.StartsWith(tagPrefix) {
			return &v
		}
	}
	return nil
}

// GetLast gets the last tag in tags that matches the prefix, see [T.StartsWith]
func (t T) GetLast(tagPrefix []string) *tag.T {
	for i := len(t) - 1; i >= 0; i-- {
		v := t[i]
		if v.StartsWith(tagPrefix) {
			return &v
		}
	}
	return nil
}

// GetAll gets all the tags that match the prefix, see [T.StartsWith]
func (t T) GetAll(tagPrefix ...string) T {
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

// AppendUnique appends a tag if it doesn't exist yet, otherwise does nothing.
// the uniqueness comparison is done based only on the first 2 elements of the
// tag.
func (t T) AppendUnique(tag tag.T) T {
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
		return errors.New("couldn'tag scan tag, it's not a json string")
	}
	err = json.Unmarshal(jtags, &t)
	chk.E(err)
	return
}

// ContainsAny returns true if any of the strings given in `values` matches any
// of the tag elements.
func (t T) ContainsAny(tagName string, values ...string) bool {
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

func (t T) String() string {
	buf := new(bytes.Buffer)
	buf.WriteByte('[')
	last := len(t) - 1
	for i := range t {
		_, _ = fmt.Fprint(buf, t[i])
		if i < last {
			buf.WriteByte(',')
		}
	}
	buf.WriteByte(']')
	return buf.String()
}
