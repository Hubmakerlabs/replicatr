package nip10

import (
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/tags"
)

func GetThreadRoot(tags tags.Tags) *tags.Tag {
	for _, tag := range tags {
		if len(tag) >= 4 && tag[0] == "e" && tag[3] == "root" {
			return &tag
		}
	}

	return tags.GetFirst([]string{"e", ""})
}

func GetImmediateReply(t tags.Tags) *tags.Tag {
	var root *tags.Tag
	var lastE *tags.Tag

	for i := 0; i <= len(t)-1; i++ {
		tag := t[i]

		if len(tag) < 2 {
			continue
		}
		if tag[0] != "e" {
			continue
		}

		if len(tag) >= 4 {
			if tag[3] == "reply" {
				return &tag
			}
			if tag[3] == "root" {
				// will be used as our first fallback
				root = &tag
				continue
			}
			if tag[3] == "mention" {
				// this invalidates this tag as a second fallback mechanism (clients that don't add markers)
				continue
			}
		}

		lastE = &tag // will be used as our second fallback (clients that don't add markers)
	}

	// if we reached this point we don't have a "reply", but if we have a "root"
	// that means this event is a direct reply to the root
	if root != nil {
		return root
	}

	// if we reached this point and we have at least one "e" we'll use that (the last)
	return lastE
}
