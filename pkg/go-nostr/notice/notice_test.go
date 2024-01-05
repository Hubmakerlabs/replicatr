package notice

import (
	"encoding/json"
	"testing"
)

func TestNoticeEnvelopeEncodingAndDecoding(t *testing.T) {
	src := `["NOTICE","kjasbdlasvdluiasvd\"kjasbdksab\\d"]`
	var env Envelope
	json.Unmarshal([]byte(src), &env)
	if env != "kjasbdlasvdluiasvd\"kjasbdksab\\d" {
		t.Error("failed to decode NOTICE")
	}

	if res, _ := json.Marshal(env); string(res) != src {
		t.Errorf("failed to encode NOTICE: expected '%s', got '%s'", src, string(res))
	}
}
