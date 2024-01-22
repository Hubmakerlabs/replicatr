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
	b, e := json.Marshal(filtertest.D)
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

	var thing filters.T
	if e = json.Unmarshal(b, &thing); e != nil {
		t.Fatalf("error: %s", e.Error())
	}
	b = thing.ToArray().Bytes()
	t.Log("original", filtertest.D)
	t.Log("mangled", thing)
	var c []byte
	c, e = json.Marshal(filtertest.D)
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
