package filtertest

import (
	"time"

	"mleku.dev/git/nostr/filter"
	"mleku.dev/git/nostr/filters"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/kinds"
	"mleku.dev/git/nostr/tag"
	"mleku.dev/git/nostr/timestamp"
)

var ten = 10

var D = filters.T{
	{
		IDs: tag.T{
			"92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
			"92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
			"92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
			"92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
		},
		Kinds: kinds.T{
			kind.TextNote, kind.MemoryHole, kind.CategorizedBookmarksList,
		},
		Authors: []string{
			"e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
			"e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
			"e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
		},
		Tags: filter.TagMap{
			"#e": {"one", "two", "three"},
			"#p": {"one", "two", "three"},
		},
		Since:  timestamp.T(time.Now().Unix() - (60 * 60)).Ptr(),
		Until:  timestamp.Now().Ptr(),
		Limit:  &ten,
		Search: "some search] terms} with bogus ]brrackets and }braces and \\\" escaped quotes \"",
	},
	{
		Kinds: []kind.T{
			kind.TextNote, kind.MemoryHole, kind.CategorizedBookmarksList,
		},
		Tags: filter.TagMap{
			"#e": {"one", "two", "three"},
			"#A": {"one", "two", "three"},
			"#x": {"one", "two", "three"},
			"#g": {"one", "two", "three"},
		},
		Until: timestamp.Now().Ptr(),
	},
}
