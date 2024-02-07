package filter_test

import (
	"encoding/json"
	"testing"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters/filtertest"
)

func TestFilterString(t *testing.T) {
	// check that array stringer and json.Marshal produce identical outputs
	a := filtertest.D.ToArray().Bytes()
	b, err := json.Marshal(filtertest.D)
	if err != nil {
		t.Fatal(err)
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

	var thing filters.T
	if err = json.Unmarshal(b, &thing); err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	b = thing.ToArray().Bytes()
	t.Log("original", filtertest.D)
	t.Log("mangled", thing)
	var c []byte
	c, err = json.Marshal(filtertest.D)
	if err != nil {
		t.Fatal(err)
	}
	for i := range a {
		if a[i] != c[i] {
			t.Fatalf("difference between outputs at index %d\n%s\n%s",
				i, a[i:], c[i:])
		}
	}

}
