package OK

import (
	"encoding/json"
	"testing"
)

func TestOKEnvelopeEncodingAndDecoding(t *testing.T) {
	okEnvelopes := []string{
		`["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",false,"error: could not connect to the database"]`,
		`["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",true,""]`,
	}

	for _, raw := range okEnvelopes {
		var env OKEnvelope
		if err := json.Unmarshal([]byte(raw), &env); err != nil {
			t.Errorf("failed to parse ok envelope json: %v", err)
		}

		asjson, err := json.Marshal(env)
		if err != nil {
			t.Errorf("failed to re marshal ok as json: %v", err)
		}

		if string(asjson) != raw {
			t.Log(string(asjson))
			t.Error("json serialization broken")
		}
	}
}
