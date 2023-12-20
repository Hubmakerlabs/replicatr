package nip1

import (
	"fmt"
	"github.com/nostric/replicatr/pkg/wire/array"
	"github.com/nostric/replicatr/pkg/wire/text"
	"reflect"
)

// CloseEnvelope is a wrapper for a signal to cancel a subscription.
type CloseEnvelope struct {
	SubscriptionID
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *CloseEnvelope) Label() (l Label) { return LClose }

func (E *CloseEnvelope) ToArray() (a array.T) {
	return array.T{CLOSE, E.SubscriptionID}
}

func (E *CloseEnvelope) String() (s string) {
	return E.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *CloseEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *CloseEnvelope) Unmarshal(buf *text.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	log.I.Ln(reflect.TypeOf(E))
	return
}
