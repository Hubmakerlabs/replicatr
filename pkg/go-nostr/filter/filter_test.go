package filter

import (
	"encoding/json"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/timestamp"
	"golang.org/x/exp/slices"
)

func TestFilterUnmarshal(t *testing.T) {
	raw := `{"ids": ["abc"],"#e":["zzz"],"#something":["nothing","bab"],"since":1644254609,"search":"test"}`
	var f Filter
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		t.Errorf("failed to parse filter json: %v", err)
	}

	if f.Since == nil || f.Since.Time().UTC().Format("2006-01-02") != "2022-02-07" ||
		f.Until != nil ||
		f.Tags == nil || len(f.Tags) != 2 || !slices.Contains(f.Tags["something"], "bab") ||
		f.Search != "test" {
		t.Error("failed to parse filter correctly")
	}
}

func TestFilterMarshal(t *testing.T) {
	until := timestamp.Timestamp(12345678)
	filterj, err := json.Marshal(Filter{
		Kinds: []int{event.KindTextNote, event.KindRecommendServer, event.KindEncryptedDirectMessage},
		Tags:  TagMap{"fruit": {"banana", "mango"}},
		Until: &until,
	})
	if err != nil {
		t.Errorf("failed to marshal filter json: %v", err)
	}

	expected := `{"kinds":[1,2,4],"until":12345678,"#fruit":["banana","mango"]}`
	if string(filterj) != expected {
		t.Errorf("filter json was wrong: %s != %s", string(filterj), expected)
	}
}

func TestFilterMatchingLive(t *testing.T) {
	var f Filter
	var evt event.T

	json.Unmarshal([]byte(`{"kinds":[1],"authors":["a8171781fd9e90ede3ea44ddca5d3abf828fe8eedeb0f3abb0dd3e563562e1fc","1d80e5588de010d137a67c42b03717595f5f510e73e42cfc48f31bae91844d59","ed4ca520e9929dfe9efdadf4011b53d30afd0678a09aa026927e60e7a45d9244"],"since":1677033299}`), &f)
	json.Unmarshal([]byte(`{"id":"5a127c9c931f392f6afc7fdb74e8be01c34035314735a6b97d2cf360d13cfb94","pubkey":"1d80e5588de010d137a67c42b03717595f5f510e73e42cfc48f31bae91844d59","created_at":1677033299,"kind":1,"tags":[["t","japan"]],"content":"If you like my art,I'd appreciate a coin or two!!\nZap is welcome!! Thanks.\n\n\n#japan #bitcoin #art #bananaart\nhttps://void.cat/d/CgM1bzDgHUCtiNNwfX9ajY.webp","sig":"828497508487ca1e374f6b4f2bba7487bc09fccd5cc0d1baa82846a944f8c5766918abf5878a580f1e6615de91f5b57a32e34c42ee2747c983aaf47dbf2a0255"}`), &evt)

	if !f.Matches(&evt) {
		t.Error("live filter should match")
	}
}

func TestFilterEquality(t *testing.T) {
	if !FilterEqual(
		Filter{Kinds: []int{event.KindEncryptedDirectMessage, event.KindDeletion}},
		Filter{Kinds: []int{event.KindEncryptedDirectMessage, event.KindDeletion}},
	) {
		t.Error("kinds filters should be equal")
	}

	if !FilterEqual(
		Filter{Kinds: []int{event.KindEncryptedDirectMessage, event.KindDeletion}, Tags: TagMap{"letter": {"a", "b"}}},
		Filter{Kinds: []int{event.KindEncryptedDirectMessage, event.KindDeletion}, Tags: TagMap{"letter": {"b", "a"}}},
	) {
		t.Error("kind+tags filters should be equal")
	}

	tm := timestamp.Now()
	if !FilterEqual(
		Filter{
			Kinds: []int{event.KindEncryptedDirectMessage, event.KindDeletion},
			Tags:  TagMap{"letter": {"a", "b"}, "fruit": {"banana"}},
			Since: &tm,
			IDs:   []string{"aaaa", "bbbb"},
		},
		Filter{
			Kinds: []int{event.KindDeletion, event.KindEncryptedDirectMessage},
			Tags:  TagMap{"letter": {"a", "b"}, "fruit": {"banana"}},
			Since: &tm,
			IDs:   []string{"aaaa", "bbbb"},
		},
	) {
		t.Error("kind+2tags+since+ids filters should be equal")
	}

	if FilterEqual(
		Filter{Kinds: []int{event.KindTextNote, event.KindEncryptedDirectMessage, event.KindDeletion}},
		Filter{Kinds: []int{event.KindEncryptedDirectMessage, event.KindDeletion, event.KindRepost}},
	) {
		t.Error("kinds filters shouldn't be equal")
	}
}

func TestFilterClone(t *testing.T) {
	ts := timestamp.Now() - 60*60
	flt := Filter{
		Kinds: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		Tags:  TagMap{"letter": {"a", "b"}, "fruit": {"banana"}},
		Since: &ts,
		IDs:   []string{"9894b4b5cb5166d23ee8899a4151cf0c66aec00bde101982a13b8e8ceb972df9"},
	}
	clone := flt.Clone()
	if !FilterEqual(flt, clone) {
		t.Errorf("clone is not equal:\n %v !=\n %v", flt, clone)
	}

	clone1 := flt.Clone()
	clone1.IDs = append(clone1.IDs, "88f0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	if FilterEqual(flt, clone1) {
		t.Error("modifying the clone ids should cause it to not be equal anymore")
	}

	clone2 := flt.Clone()
	clone2.Tags["letter"] = append(clone2.Tags["letter"], "c")
	if FilterEqual(flt, clone2) {
		t.Error("modifying the clone tag items should cause it to not be equal anymore")
	}

	clone3 := flt.Clone()
	clone3.Tags["g"] = []string{"drt"}
	if FilterEqual(flt, clone3) {
		t.Error("modifying the clone tag map should cause it to not be equal anymore")
	}

	clone4 := flt.Clone()
	*clone4.Since++
	if FilterEqual(flt, clone4) {
		t.Error("modifying the clone since should cause it to not be equal anymore")
	}
}
