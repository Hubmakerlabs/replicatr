package nostr

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

type EventEnvelope struct {
	SubscriptionID *string
	event.T
}

var _ Envelope = (*EventEnvelope)(nil)

func (_ EventEnvelope) Label() string { return "EVENT" }

func (v *EventEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	switch len(arr) {
	case 2:
		return easyjson.Unmarshal([]byte(arr[1].Raw), &v.T)
	case 3:
		v.SubscriptionID = &arr[1].Str
		return easyjson.Unmarshal([]byte(arr[2].Raw), &v.T)
	default:
		return fmt.Errorf("failed to decode EVENT envelope")
	}
}

func (v EventEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["EVENT",`)
	if v.SubscriptionID != nil {
		w.RawString(`"` + *v.SubscriptionID + `",`)
	}
	v.MarshalEasyJSON(&w)
	w.RawString(`]`)
	return w.BuildBytes()
}
