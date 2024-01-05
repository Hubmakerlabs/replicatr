package eose

import (
	"encoding/json"
	"testing"
)

func TestEoseEnvelopeEncodingAndDecoding(t *testing.T) {
	src := `["EOSE","kjasbdlasvdluiasvd\"kjasbdksab\\d"]`
	var env Envelope
	json.Unmarshal([]byte(src), &env)
	if env != "kjasbdlasvdluiasvd\"kjasbdksab\\d" {
		t.Error("failed to decode EOSE")
	}

	if res, _ := json.Marshal(env); string(res) != src {
		t.Errorf("failed to encode EOSE: expected '%s', got '%s'", src, string(res))
	}
}
