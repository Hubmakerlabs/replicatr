package eose

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

const RELAY = "wss://nostr.mom"

var _ envelopes.Envelope = (*EOSEEnvelope)(nil)

type EOSEEnvelope string

func (_ EOSEEnvelope) Label() string { return "EOSE" }
func (e EOSEEnvelope) String() string {
	v, _ := json.Marshal(e)
	return string(v)
}

func (v *EOSEEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 2 {
		return fmt.Errorf("failed to decode EOSE envelope")
	}
	*v = EOSEEnvelope(arr[1].Str)
	return nil
}

func (v EOSEEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["EOSE",`)
	w.Raw(json.Marshal(string(v)))
	w.RawString(`]`)
	return w.BuildBytes()
}
