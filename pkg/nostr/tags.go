package nostr

import (
	"encoding/json"
	"errors"
)

// Tags is a list of Tag - which are lists of string elements with ordering and
// no uniqueness constraint (not a set).
type Tags []Tag

// GetFirst gets the first tag in tags that matches the prefix, see [Tag.StartsWith]
func (tags Tags) GetFirst(tagPrefix []string) *Tag {
	for _, v := range tags {
		if v.StartsWith(tagPrefix) {
			return &v
		}
	}
	return nil
}

// GetLast gets the last tag in tags that matches the prefix, see [Tag.StartsWith]
func (tags Tags) GetLast(tagPrefix []string) *Tag {
	for i := len(tags) - 1; i >= 0; i-- {
		v := tags[i]
		if v.StartsWith(tagPrefix) {
			return &v
		}
	}
	return nil
}

// GetAll gets all the tags that match the prefix, see [Tag.StartsWith]
func (tags Tags) GetAll(tagPrefix []string) Tags {
	result := make(Tags, 0, len(tags))
	for _, v := range tags {
		if v.StartsWith(tagPrefix) {
			result = append(result, v)
		}
	}
	return result
}

// FilterOut removes all tags that match the prefix, see [Tag.StartsWith]
func (tags Tags) FilterOut(tagPrefix []string) Tags {
	filtered := make(Tags, 0, len(tags))
	for _, v := range tags {
		if !v.StartsWith(tagPrefix) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// AppendUnique appends a tag if it doesn't exist yet, otherwise does nothing.
// the uniqueness comparison is done based only on the first 2 elements of the tag.
func (tags Tags) AppendUnique(tag Tag) Tags {
	n := len(tag)
	if n > 2 {
		n = 2
	}
	if tags.GetFirst(tag[:n]) == nil {
		return append(tags, tag)
	}
	return tags
}

// Scan parses a string or raw bytes that should be a string and embeds the
// values into the tags variable from which this method is invoked.
func (tags Tags) Scan(src any) (err error) {
	var jtags []byte
	switch v := src.(type) {
	case []byte:
		jtags = v
	case string:
		jtags = []byte(v)
	default:
		return errors.New("couldn't scan tags, it's not a json string")
	}
	err = json.Unmarshal(jtags, &tags)
	return
}

// ContainsAny returns true if any of the strings given in `values` matches any
// of the tag elements.
func (tags Tags) ContainsAny(tagName string, values []string) bool {
	for _, v := range tags {
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

// MarshalTo appends the JSON encoded byte of Tags as [][]string to dst. String
// escaping is as described in RFC8259.
func (tags Tags) MarshalTo(dst []byte) []byte {
	dst = append(dst, '[')
	for i, tag := range tags {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = tag.marshalTo(dst)
	}
	dst = append(dst, ']')
	return dst
}
