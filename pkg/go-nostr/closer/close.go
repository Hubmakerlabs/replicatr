package closer

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

var _ envelopes.E = (*Envelope)(nil)

type Envelope string

func (_ *Envelope) Label() string { return "CLOSE" }
func (c *Envelope) String() string {
	v, _ := json.Marshal(c)
	return string(v)
}

func (c *Envelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	switch len(arr) {
	case 2:
		*c = Envelope(arr[1].Str)
		return nil
	default:
		return fmt.Errorf("failed to decode CLOSE envelope")
	}
}

func (c *Envelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["CLOSE",`)
	w.Raw(json.Marshal(string(*c)))
	w.RawString(`]`)
	return w.BuildBytes()
}
