package count

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/filters"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

var _ envelopes.Envelope = (*Envelope)(nil)

type Envelope struct {
	SubscriptionID string
	filters.T
	Count *int64
}

func (_ Envelope) Label() string { return "COUNT" }
func (c Envelope) String() string {
	v, _ := json.Marshal(c)
	return string(v)
}

func (v *Envelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 3 {
		return fmt.Errorf("failed to decode COUNT envelope: missing filters")
	}
	v.SubscriptionID = arr[1].Str

	if len(arr) < 3 {
		return fmt.Errorf("COUNT array must have at least 3 items")
	}

	var countResult struct {
		Count *int64 `json:"count"`
	}
	if e := json.Unmarshal([]byte(arr[2].Raw), &countResult); e == nil && countResult.Count != nil {
		v.Count = countResult.Count
		return nil
	}

	v.T = make(filters.T, len(arr)-2)
	f := 0
	for i := 2; i < len(arr); i++ {
		item := []byte(arr[i].Raw)

		if e := easyjson.Unmarshal(item, &v.T[f]); e != nil {
			return fmt.Errorf("%w -- on filter %d", e, f)
		}

		f++
	}

	return nil
}

func (v Envelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["COUNT",`)
	w.RawString(`"` + v.SubscriptionID + `"`)
	if v.Count != nil {
		w.RawString(fmt.Sprintf(`{"count":%d}`, *v.Count))
	} else {
		for _, f := range v.T {
			w.RawString(`,`)
			f.MarshalEasyJSON(&w)
		}
	}
	w.RawString(`]`)
	return w.BuildBytes()
}
