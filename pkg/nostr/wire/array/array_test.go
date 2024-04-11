package array

import (
	"encoding/json"
	"testing"
	"time"
)

var literal = T{"1", "aoeu", "3", time.Now(), "sorta normal", 0.333}

func TestArray(t *testing.T) {
	var b []byte
	var err error
	b, err = json.Marshal(literal)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
	t.Log(literal)
}
