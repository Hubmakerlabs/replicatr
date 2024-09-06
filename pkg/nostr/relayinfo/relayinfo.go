package relayinfo

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/object"
)

func (ri *T) ToObject() (o object.T) {
	return object.T{
		{"name", ri.Name},
		{"description", ri.Description},
		{"pubkey", ri.PubKey},
		{"contact", ri.Contact},
		{"supported_nips", ri.Nips},
		{"software", ri.Software},
		{"version", ri.Version},
		{"limitation", ri.Limitation},
		{"relay_countries", ri.RelayCountries},
		{"language_tags", ri.LanguageTags},
		{"tags", ri.Tags},
		{"posting_policy", ri.PostingPolicy},
		{"payments_url", ri.PaymentsURL},
		{"fees", ri.Fees},
		{"icon", ri.Icon},
	}
}

// AddSupportedNIP appends a supported NIP number to a RelayInfo.
func (ri *T) AddSupportedNIP(n int) {
	idx, exists := ri.Nips.HasNumber(n)
	if exists {
		return
	}
	ri.Nips = append(ri.Nips, -1)
	copy(ri.Nips[idx+1:], ri.Nips[idx:])
	ri.Nips[idx] = n
}

func (ri *Limits) ToObject() (o object.T) {
	return object.T{
		{"max_message_length,omitempty", ri.MaxMessageLength},
		{"max_subscriptions,omitempty", ri.MaxSubscriptions},
		{"max_filters,omitempty", ri.MaxFilters},
		{"max_limit,omitempty", ri.MaxLimit},
		{"max_subid_length,omitempty", ri.MaxSubidLength},
		{"max_event_tags,omitempty", ri.MaxEventTags},
		{"max_content_length,omitempty", ri.MaxContentLength},
		{"min_pow_difficulty,omitempty", ri.MinPowDifficulty},
		{"auth_required", ri.AuthRequired},
		{"payment_required", ri.PaymentRequired},
		{"restricted_writes", ri.RestrictedWrites},
	}
}

type Admission struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
}

type Subscription struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
	Period int    `json:"period"`
}

type Publication []struct {
	Kinds  []int  `json:"kinds"`
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
}

// Fees defines the fee structure used for a paid relay.
type Fees struct {
	Admission    []Admission    `json:"admission,omitempty"`
	Subscription []Subscription `json:"subscription,omitempty"`
	Publication  []Publication  `json:"publication,omitempty"`
}
