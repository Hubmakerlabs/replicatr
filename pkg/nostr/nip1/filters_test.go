package nip1_test

import (
	"encoding/json"
	"mleku.online/git/replicatr/pkg/nostr/kind"
	"mleku.online/git/replicatr/pkg/nostr/nip1"
	"mleku.online/git/replicatr/pkg/nostr/tag"
	"mleku.online/git/replicatr/pkg/nostr/timestamp"
	"testing"
	"time"
)

var filt = nip1.Filters{
	{
		IDs: tag.T{
			"92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
			"92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
			"92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
			"92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
		},
		Kinds: kind.Array{
			kind.TextNote, kind.MemoryHole, kind.CategorizedBookmarksList,
		},
		Authors: []string{
			"e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
			"e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
			"e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
		},
		Tags: nip1.TagMap{
			"#e": {"one", "two", "three"},
			"#p": {"one", "two", "three"},
		},
		Since:  timestamp.T(time.Now().Unix() - (60 * 60)).Ptr(),
		Until:  timestamp.Now().Ptr(),
		Limit:  10,
		Search: "some search terms}",
	},
	{
		Kinds: []kind.T{
			kind.TextNote, kind.MemoryHole, kind.CategorizedBookmarksList,
		},
		Tags: nip1.TagMap{
			"#e": {"one", "two", "three"},
			"#A": {"one", "two", "three"},
			"#x": {"one", "two", "three"},
			"#g": {"one", "two", "three"},
		},
		Until: timestamp.Now().Ptr(),
	},
}

func TestFilterString(t *testing.T) {
	// check that array stringer and json.Marshal produce identical outputs
	a := filt.ToArray().Bytes()
	b, e := json.Marshal(filt)
	if e != nil {
		t.Fatal(e)
	}
	if len(a) != len(b) {
		t.Fatal("outputs not the same length")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("difference between outputs at index %d\n%s\n%s",
				i, a[i:], b[i:])
		}
	}
	// check that unmarshalling this back to the runtime form produces
	// completely equal data.

	var thing nip1.Filters
	if e = json.Unmarshal(b, &thing); e != nil {
		t.Fatalf("error: %s", e.Error())
	}
	b = thing.ToArray().Bytes()
	t.Log("original", filt)
	t.Log("mangled", thing)
	var c []byte
	c, e = json.Marshal(filt)
	if e != nil {
		t.Fatal(e)
	}
	for i := range a {
		if a[i] != c[i] {
			t.Fatalf("difference between outputs at index %d\n%s\n%s",
				i, a[i:], c[i:])
		}
	}

}
