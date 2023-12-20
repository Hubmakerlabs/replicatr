package nip1

import (
	"fmt"
	"github.com/nostric/replicatr/pkg/wire/array"
	"github.com/nostric/replicatr/pkg/wire/text"
	"reflect"
)

// EOSEEnvelope is a message that indicates that all cached events have been
// delivered and thereafter events will be new and delivered in pubsub subscribe
// fashion while the socket remains open.
type EOSEEnvelope struct {
	SubscriptionID
}

// Label returns the label enum/type of the envelope. The relevant bytes could
// be retrieved using nip1.Labels[Label]
func (E *EOSEEnvelope) Label() (l Label) { return LEOSE }

func (E *EOSEEnvelope) ToArray() (a array.T) {
	a = array.T{EOSE, E.SubscriptionID}
	return
}

func (E *EOSEEnvelope) String() (s string) {
	return E.ToArray().String()
}

// MarshalJSON returns the JSON encoded form of the envelope.
func (E *EOSEEnvelope) MarshalJSON() (bytes []byte, e error) {
	return E.ToArray().Bytes(), nil
}

// Unmarshal the envelope.
func (E *EOSEEnvelope) Unmarshal(buf *text.Buffer) (e error) {
	if E == nil {
		return fmt.Errorf("cannot unmarshal to nil pointer")
	}
	log.I.Ln(reflect.TypeOf(E))
	return
}
