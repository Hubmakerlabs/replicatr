package notice

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

var _ envelopes.E = (*Envelope)(nil)

type Envelope string

func (_ Envelope) Label() string { return "NOTICE" }
func (n Envelope) String() string {
	v, _ := json.Marshal(n)
	return string(v)
}

func (v *Envelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 2 {
		return fmt.Errorf("failed to decode NOTICE envelope")
	}
	*v = Envelope(arr[1].Str)
	return nil
}

func (v Envelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["NOTICE",`)
	w.Raw(json.Marshal(string(v)))
	w.RawString(`]`)
	return w.BuildBytes()
}
