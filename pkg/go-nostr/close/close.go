package close

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

var _ envelopes.Envelope = (*CloseEnvelope)(nil)

type CloseEnvelope string

func (_ CloseEnvelope) Label() string { return "CLOSE" }
func (c CloseEnvelope) String() string {
	v, _ := json.Marshal(c)
	return string(v)
}

func (v *CloseEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	switch len(arr) {
	case 2:
		*v = CloseEnvelope(arr[1].Str)
		return nil
	default:
		return fmt.Errorf("failed to decode CLOSE envelope")
	}
}

func (v CloseEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["CLOSE",`)
	w.Raw(json.Marshal(string(v)))
	w.RawString(`]`)
	return w.BuildBytes()
}
