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
		var env Envelope
		if e := json.Unmarshal([]byte(raw), &env); e != nil {
			t.Errorf("failed to parse ok envelope json: %v", e)
		}

		asjson, e := json.Marshal(env)
		if e != nil {
			t.Errorf("failed to re marshal ok as json: %v", e)
		}

		if string(asjson) != raw {
			t.Log(string(asjson))
			t.Error("json serialization broken")
		}
	}
}
