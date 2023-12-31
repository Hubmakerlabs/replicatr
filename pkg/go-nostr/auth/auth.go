package auth

import (
	"encoding/json"
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/envelopes"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

var _ envelopes.E = (*Envelope)(nil)

type Envelope struct {
	Challenge *string
	Event     event.T
}

func (_ Envelope) Label() string { return "AUTH" }
func (a Envelope) String() string {
	v, _ := json.Marshal(a)
	return string(v)
}

func (v *Envelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 2 {
		return fmt.Errorf("failed to decode Auth envelope: missing fields")
	}
	if arr[1].IsObject() {
		return easyjson.Unmarshal([]byte(arr[1].Raw), &v.Event)
	} else {
		v.Challenge = &arr[1].Str
	}
	return nil
}

func (v Envelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	w.RawString(`["AUTH",`)
	if v.Challenge != nil {
		w.Raw(json.Marshal(*v.Challenge))
	} else {
		v.Event.MarshalEasyJSON(&w)
	}
	w.RawString(`]`)
	return w.BuildBytes()
}
