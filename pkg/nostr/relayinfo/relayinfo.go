package relayinfo

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/number"
	"github.com/Hubmakerlabs/replicatr/pkg/wire/object"
)

// T provides the information for a relay on the network as regards to
// versions, NIP support, contact, policies, and payment requirements.
//
// todo: change the string slices into tag type
type T struct {
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	PubKey         string      `json:"pubkey"`
	Contact        string      `json:"contact"`
	SupportedNIPs  number.List `json:"supported_nips"`
	Software       string      `json:"software"`
	Version        string      `json:"version"`
	Limitation     *Limits     `json:"limitation,omitempty"`
	RelayCountries []string    `json:"relay_countries,omitempty"`
	LanguageTags   []string    `json:"language_tags,omitempty"`
	Tags           []string    `json:"tags,omitempty"`
	PostingPolicy  string      `json:"posting_policy,omitempty"`
	PaymentsURL    string      `json:"payments_url,omitempty"`
	Fees           *Fees       `json:"fees,omitempty"`
	Icon           string      `json:"icon"`
}

func (ri *T) ToObject() (o object.T) {
	return object.T{
		{"name", ri.Name},
		{"description", ri.Description},
		{"pubkey", ri.PubKey},
		{"contact", ri.Contact},
		{"supported_nips", ri.SupportedNIPs},
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
	idx, exists := ri.SupportedNIPs.HasNumber(n)
	if exists {
		return
	}
	ri.SupportedNIPs = append(ri.SupportedNIPs, -1)
	copy(ri.SupportedNIPs[idx+1:], ri.SupportedNIPs[idx:])
	ri.SupportedNIPs[idx] = n
}

// Limits specifies the various restrictions and limitations that apply to
// interactions with a given relay.
type Limits struct {
	MaxMessageLength int  `json:"max_message_length,omitempty"`
	MaxSubscriptions int  `json:"max_subscriptions,omitempty"`
	MaxFilters       int  `json:"max_filters,omitempty"`
	MaxLimit         int  `json:"max_limit,omitempty"`
	MaxSubidLength   int  `json:"max_subid_length,omitempty"`
	MaxEventTags     int  `json:"max_event_tags,omitempty"`
	MaxContentLength int  `json:"max_content_length,omitempty"`
	MinPowDifficulty int  `json:"min_pow_difficulty,omitempty"`
	AuthRequired     bool `json:"auth_required"`
	PaymentRequired  bool `json:"payment_required"`
	RestrictedWrites bool `json:"restricted_writes"`
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

// Fees defines the fee structure used for a paid relay.
type Fees struct {
	Admission []struct {
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
	} `json:"admission,omitempty"`
	Subscription []struct {
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
		Period int    `json:"period"`
	} `json:"subscription,omitempty"`
	Publication []struct {
		Kinds  []int  `json:"kinds"`
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
	} `json:"publication,omitempty"`
}
