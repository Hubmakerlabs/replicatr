package closed

import (
	"encoding/json"
	"testing"
)

func TestClosedEnvelopeEncodingAndDecoding(t *testing.T) {
	for _, src := range []string{
		`["CLOSED","_","error: something went wrong"]`,
		`["CLOSED",":1","auth-required: take a selfie and send it to the CIA"]`,
	} {
		var env ClosedEnvelope
		json.Unmarshal([]byte(src), &env)
		if env.SubscriptionID != "_" && env.SubscriptionID != ":1" {
			t.Error("failed to decode CLOSED")
		}
		if res, _ := json.Marshal(env); string(res) != src {
			t.Errorf("failed to encode CLOSED: expected '%s', got '%s'", src, string(res))
		}
	}
}
