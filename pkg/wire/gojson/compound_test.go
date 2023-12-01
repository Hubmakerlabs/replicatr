package array

import (
	"encoding/json"
	"mleku.online/git/replicatr/pkg/wire/array"
	"mleku.online/git/replicatr/pkg/wire/object"
	"testing"
	"time"
)

var literal = object.T{
	{"1", "aoeu"},
	{"3", time.Now()},
	{"sorta normal", 0.333},
	{"array_with_object_inside", array.T{
		"1",
		"aoeu",
		"3",
		object.T{
			{"1", "aoeu"},
			{"3", time.Now()},
			{"sorta normal", 0.333},
			{"11", "aoeu"},
			{"13", time.Now()},
			{"1sorta normal", 0.333},
		},
		time.Now(),
		"sorta normal",
		0.333,
	}},
}

var literalAsMapStringInterface = map[string]interface{}{
	"1":            "aoeu",
	"3":            time.Now(),
	"sorta normal": 0.333,
	"array_with_object_inside": array.T{
		"1",
		"aoeu",
		"3",
		map[string]interface{}{
			"1":             "aoeu",
			"3":             time.Now(),
			"sorta normal":  0.333,
			"11":            "aoeu",
			"13":            time.Now(),
			"1sorta normal": 0.333,
		},
		time.Now(),
		"sorta normal",
		0.333,
	},
}

func TestObject(t *testing.T) {

	// This demonstrates the mutual embedding of array.T and object.T with
	// object.T's order respecting properties.

	var b []byte
	var e error
	b, e = json.Marshal(literalAsMapStringInterface)
	if e != nil {
		t.Fatal(e)
	}
	t.Log(string(b)) // how this looks using the encoding/json map[string]interface{} convention
	b, e = json.Marshal(literal)
	if e != nil {
		t.Fatal(e)
	}
	t.Log(string(b)) // just to show the underlying structure that makes K/V pairs as mangled by encoding/json.

	// this version preserves ordering in the object.T parts where the map[string]interface{} ordering is lost.
	t.Log(literal)
}
