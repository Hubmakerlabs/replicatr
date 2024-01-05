package req

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

var _ envelopes.E = (*Envelope)(nil)

type Envelope struct {
	SubscriptionID string
	filters.T
}

func (_ Envelope) Label() string { return "REQ" }

func (v *Envelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 3 {
		return fmt.Errorf("failed to decode REQ envelope: missing filters")
	}
	v.SubscriptionID = arr[1].Str
	v.T = make(filters.T, len(arr)-2)
	f := 0
	for i := 2; i < len(arr); i++ {
		if e := easyjson.Unmarshal([]byte(arr[i].Raw), &v.T[f]); e != nil {
			return fmt.Errorf("%w -- on filter %d", e, f)
		}
		f++
	}

	return nil
}

func (v Envelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["REQ",`)
	w.RawString(`"` + v.SubscriptionID + `"`)
	for _, f := range v.T {
		w.RawString(`,`)
		f.MarshalEasyJSON(&w)
	}
	w.RawString(`]`)
	return w.BuildBytes()
}
