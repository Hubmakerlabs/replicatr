package nostr

// RelayInfo provides the information for a relay on the network as regards to
// versions, NIP support, contact, policies, and payment requirements.
type RelayInfo struct {
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	PubKey         string       `json:"pubkey"`
	Contact        string       `json:"contact"`
	SupportedNIPs  NList        `json:"supported_nips"`
	Software       string       `json:"software"`
	Version        string       `json:"version"`
	Limitation     *RelayLimits `json:"limitation,omitempty"`
	RelayCountries []string     `json:"relay_countries,omitempty"`
	LanguageTags   []string     `json:"language_tags,omitempty"`
	Tags           []string     `json:"tags,omitempty"`
	PostingPolicy  string       `json:"posting_policy,omitempty"`
	PaymentsURL    string       `json:"payments_url,omitempty"`
	Fees           *RelayFees   `json:"fees,omitempty"`
	Icon           string       `json:"icon"`
}

// AddSupportedNIP appends a supported NIP number to a RelayInfo.
func (info *RelayInfo) AddSupportedNIP(number int) {
	idx, exists := info.SupportedNIPs.HasNumber(number)
	if exists {
		return
	}
	info.SupportedNIPs = append(info.SupportedNIPs, -1)
	copy(info.SupportedNIPs[idx+1:], info.SupportedNIPs[idx:])
	info.SupportedNIPs[idx] = number
}

// RelayLimits specifies the various restrictions and limitations that apply to
// interactions with a given relay.
type RelayLimits struct {
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

// RelayFees defines the fee structure used for a paid relay.
type RelayFees struct {
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
