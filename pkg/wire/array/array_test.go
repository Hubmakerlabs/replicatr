package array

import (
	"encoding/json"
	"testing"
	"time"
)

var literal = T{"1", "aoeu", "3", time.Now(), "sorta normal", 0.333}

func TestArray(t *testing.T) {
	var b []byte
	var e error
	b, e = json.Marshal(literal)
	if e != nil {
		t.Fatal(e)
	}
	t.Log(string(b))
	t.Log(literal)
}
