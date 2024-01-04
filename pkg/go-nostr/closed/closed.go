package closed

import (
	"encoding/json"
	"fmt"

	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

type Envelope struct {
	SubscriptionID string
	Reason         string
}

func (_ Envelope) Label() string { return "CLOSED" }

func (c Envelope) String() string {
	v, _ := json.Marshal(c)
	return string(v)
}

func (v *Envelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	switch len(arr) {
	case 3:
		*v = Envelope{arr[1].Str, arr[2].Str}
		return nil
	default:
		return fmt.Errorf("failed to decode CLOSED envelope")
	}
}

func (v Envelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["CLOSED",`)
	w.Raw(json.Marshal(string(v.SubscriptionID)))
	w.RawString(`,`)
	w.Raw(json.Marshal(v.Reason))
	w.RawString(`]`)
	return w.BuildBytes()
}
