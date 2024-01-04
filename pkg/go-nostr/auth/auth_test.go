package auth

import (
	"encoding/json"
	"testing"
)

func TestAuthEnvelopeEncodingAndDecoding(t *testing.T) {
	authEnvelopes := []string{
		`["AUTH","kjsabdlasb aslkd kasndkad \"as.kdnbskadb"]`,
		`["AUTH",{"id":"ae1fc7154296569d87ca4663f6bdf448c217d1590d28c85d158557b8b43b4d69","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","created_at":1683660344,"kind":1,"tags":[],"content":"hello world","sig":"94e10947814b1ebe38af42300ecd90c7642763896c4f69506ae97bfdf54eec3c0c21df96b7d95daa74ff3d414b1d758ee95fc258125deebc31df0c6ba9396a51"}]`,
	}

	for _, raw := range authEnvelopes {
		var env AuthEnvelope
		if err := json.Unmarshal([]byte(raw), &env); err != nil {
			t.Errorf("failed to parse auth envelope json: %v", err)
		}

		asjson, err := json.Marshal(env)
		if err != nil {
			t.Errorf("failed to re marshal auth as json: %v", err)
		}

		if string(asjson) != raw {
			t.Log(string(asjson))
			t.Error("json serialization broken")
		}
	}
}
