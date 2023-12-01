package array

import (
	"encoding/json"
	"mleku.online/git/replicatr/pkg/array"
	"mleku.online/git/replicatr/pkg/object"
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
		},
		time.Now(),
		"sorta normal",
		0.333,
	}},
}

func TestObject(t *testing.T) {
	var b []byte
	var e error
	b, e = json.Marshal(literal)
	if e != nil {
		t.Fatal(e)
	}
	t.Log(string(b))
	t.Log(literal)
}
