// Package envelopes is for message envelopes (relay<->relay<->client message
// wrappers) that aren't specified in NIP-1
package envelopes

import "github.com/mleku/replicatr/pkg/nostr/nip1"

const (
	LabelCount = "COUNT"
	LabelAuth  = "AUTH"
)

type CountEnvelope struct {
	SubscriptionID nip1.SubscriptionID
	Filters        nip1.Filters
	Count          *int64
}

type AuthEnvelope struct {
	Challenge *string
	Event     nip1.Event
}
