package nip11

import (
	"sync"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/wire/array"
)

type NIP struct {
	Description string
	Number      int
}

// this is the list of all nips and their titles for use in supported_nips field
var (
	BasicProtocol                  = NIP{"Basic protocol flow description", 1}
	NIP1                           = BasicProtocol
	FollowList                     = NIP{"Follow List", 2}
	NIP2                           = FollowList
	OpenTimestampsAttestations     = NIP{"OpenTimestamps Attestations for Events", 3}
	NIP3                           = OpenTimestampsAttestations
	EncryptedDirectMessage         = NIP{"Encrypted Direct Message --- unrecommended: deprecated in favor of NIP-44", 4}
	NIP4                           = EncryptedDirectMessage
	MappingNostrKeysToDNS          = NIP{"Mapping Nostr keys to DNS-based internet identifiers", 5}
	NIP5                           = MappingNostrKeysToDNS
	HandlingMentions               = NIP{"Handling Mentions --- unrecommended: deprecated in favor of NIP-27", 8}
	NIP8                           = HandlingMentions
	EventDeletion                  = NIP{"Event Deletion", 9}
	NIP9                           = EventDeletion
	RelayInformationDocument       = NIP{"Relay Information Document", 11}
	NIP11                          = RelayInformationDocument
	SubjectTag                     = NIP{"Subject tag in text events", 14}
	NIP14                          = SubjectTag
	NostrMarketplace               = NIP{"Nostr Marketplace (for resilient marketplaces)", 15}
	NIP15                          = NostrMarketplace
	Reposts                        = NIP{"Reposts", 18}
	NIP18                          = Reposts
	Bech32EncodedEntities          = NIP{"bech32-encoded entities", 19}
	NIP19                          = Bech32EncodedEntities
	NostrURIScheme                 = NIP{"nostr: URI scheme", 21}
	NIP21                          = NostrURIScheme
	LongFormContent                = NIP{"Long-form Content", 23}
	NIP23                          = LongFormContent
	ExtraMetadata                  = NIP{"Extra metadata fields and tags", 24}
	NIP24                          = ExtraMetadata
	Reactions                      = NIP{"Reactions", 25}
	NIP25                          = Reactions
	DelegatedEventSigning          = NIP{"Delegated Event Signing", 26}
	NIP26                          = DelegatedEventSigning
	TextNoteReferences             = NIP{"Text Note References", 27}
	NIP27                          = TextNoteReferences
	PublicChat                     = NIP{"Public Chat", 28}
	NIP28                          = PublicChat
	CustomEmoji                    = NIP{"Custom Emoji", 30}
	NIP30                          = CustomEmoji
	Labeling                       = NIP{"Labeling", 32}
	NIP32                          = Labeling
	SensitiveContent               = NIP{"Sensitive Content", 36}
	NIP36                          = SensitiveContent
	UserStatuses                   = NIP{"User Statuses", 38}
	NIP38                          = UserStatuses
	ExternalIdentitiesInProfiles   = NIP{"External Identities in Profiles", 39}
	NIP39                          = ExternalIdentitiesInProfiles
	ExpirationTimestamp            = NIP{"Expiration Timestamp", 40}
	NIP40                          = ExpirationTimestamp
	Authentication                 = NIP{"Authentication of clients to relays", 42}
	NIP42                          = Authentication
	VersionedEncryption            = NIP{"Versioned Encryption", 44}
	NIP44                          = VersionedEncryption
	CountingResults                = NIP{"Counting results", 45}
	NIP45                          = CountingResults
	NostrConnect                   = NIP{"Nostr Connect", 46}
	NIP46                          = NostrConnect
	WalletConnect                  = NIP{"Wallet Connect", 47}
	NIP47                          = WalletConnect
	ProxyTags                      = NIP{"Proxy Tags", 48}
	NIP48                          = ProxyTags
	SearchCapability               = NIP{"Search Capability", 50}
	NIP50                          = SearchCapability
	Lists                          = NIP{"Lists", 51}
	NIP51                          = Lists
	CalendarEvents                 = NIP{"Calendar Events", 52}
	NIP52                          = CalendarEvents
	LiveActivities                 = NIP{"Live Activities", 53}
	NIP53                          = LiveActivities
	Reporting                      = NIP{"Reporting", 56}
	NIP56                          = Reporting
	LightningZaps                  = NIP{"Lightning Zaps", 57}
	NIP57                          = LightningZaps
	Badges                         = NIP{"Badges", 58}
	NIP58                          = Badges
	RelayListMetadata              = NIP{"Relay List Metadata", 65}
	NIP65                          = RelayListMetadata
	ModeratedCommunities           = NIP{"Moderated Communities", 72}
	NIP72                          = ModeratedCommunities
	ZapGoals                       = NIP{"Zap Goals", 75}
	NIP75                          = ZapGoals
	ApplicationSpecificData        = NIP{"Application-specific data", 78}
	NIP78                          = ApplicationSpecificData
	Highlights                     = NIP{"Highlights", 84}
	NIP84                          = Highlights
	RecommendedApplicationHandlers = NIP{"Recommended Application Handlers", 89}
	NIP89                          = RecommendedApplicationHandlers
	DataVendingMachines            = NIP{"Data Vending Machines", 90}
	NIP90                          = DataVendingMachines
	FileMetadata                   = NIP{"File Metadata", 94}
	NIP94                          = FileMetadata
	HTTPFileStorageIntegration     = NIP{"HTTP File Storage Integration", 96}
	NIP96                          = HTTPFileStorageIntegration
	HTTPAuth                       = NIP{"HTTP Auth", 98}
	NIP98                          = HTTPAuth
	ClassifiedListings             = NIP{"Classified Listings", 99}
	NIP99                          = ClassifiedListings
)

var NIPMap = map[int]NIP{
	1:  NIP1,
	2:  NIP2,
	3:  NIP3,
	4:  NIP4,
	5:  NIP5,
	8:  NIP8,
	9:  NIP9,
	11: NIP11,
	14: NIP14,
	15: NIP15,
	18: NIP18,
	19: NIP19,
	21: NIP21,
	23: NIP23,
	24: NIP24,
	25: NIP25,
	26: NIP26,
	27: NIP27,
	28: NIP28,
	30: NIP30,
	32: NIP32,
	36: NIP36,
	38: NIP38,
	39: NIP39,
	40: NIP40,
	42: NIP42,
	44: NIP44,
	45: NIP45,
	46: NIP46,
	47: NIP47,
	48: NIP48,
	50: NIP50,
	51: NIP51,
	52: NIP52,
	53: NIP53,
	56: NIP56,
	57: NIP57,
	58: NIP58,
	65: NIP65,
	72: NIP72,
	75: NIP75,
	78: NIP78,
	84: NIP84,
	89: NIP89,
	90: NIP90,
	94: NIP94,
	96: NIP96,
	98: NIP98,
	99: NIP99,
}

type Limits struct {
	// MaxMessageLength is the maximum number of bytes for incoming JSON
	// that the relay will attempt to decode and act upon. When you send large
	// subscriptions, you will be limited by this value. It also effectively
	// limits the maximum size of any event. Value is calculated from [ to ] and
	// is after UTF-8 serialization (so some unicode characters will cost 2-3
	// bytes). It is equal to the maximum size of the WebSocket message frame.
	MaxMessageLength int `json:"max_message_length,omitempty"`
	// MaxSubscriptions is total number of subscriptions that may be active on a
	// single websocket connection to this relay. It's possible that
	// authenticated clients with a (paid) relationship to the relay may have
	// higher limits.
	MaxSubscriptions int `json:"max_subscriptions,omitempty"`
	// MaxFilter is maximum number of filter values in each subscription. Must
	// be one or higher.
	MaxFilters int `json:"max_filters,omitempty"`
	// MaxLimit is the relay server will clamp each filter's limit value to this
	// number. This means the client won't be able to get more than this number
	// of events from a single subscription filter. This clamping is typically
	// done silently by the relay, but with this number, you can know that there
	// are additional results if you narrowed your filter's time range or other
	// parameters.
	MaxLimit int `json:"max_limit,omitempty"`
	// MaxSubidLength is the maximum length of subscription id as a string.
	MaxSubidLength int `json:"max_subid_length,omitempty"`
	// MaxEventTags in any event, this is the maximum number of elements in the
	// tags list.
	MaxEventTags int `json:"max_event_tags,omitempty"`
	// MaxContentLength maximum number of characters in the content field of any
	// event. This is a count of unicode characters. After serializing into JSON
	// it may be larger (in bytes), and is still subject to the
	// max_message_length, if defined.
	MaxContentLength int `json:"max_content_length,omitempty"`
	// MinPowDifficulty new events will require at least this difficulty of PoW,
	// based on NIP-13, or they will be rejected by this server.
	MinPowDifficulty int `json:"min_pow_difficulty,omitempty"`
	// AuthRequired means the relay requires NIP-42 authentication to happen
	// before a new connection may perform any other action. Even if set to
	// False, authentication may be required for specific actions.
	AuthRequired bool `json:"auth_required"`
	// PaymentRequired this relay requires payment before a new connection may
	// perform any action.
	PaymentRequired bool `json:"payment_required"`
	// RestrictedWrites this relay requires some kind of condition to be
	// fulfilled in order to accept events (not necessarily, but including
	// payment_required and min_pow_difficulty). This should only be set to true
	// when users are expected to know the relay policy before trying to write
	// to it -- like belonging to a special pubkey-based whitelist or writing
	// only events of a specific niche kind or content. Normal anti-spam
	// heuristics, for example, do not qualify.q
	RestrictedWrites bool        `json:"restricted_writes"`
	Oldest           timestamp.T `json:"created_at_lower_limit"`
	Newest           timestamp.T `json:"created_at_upper_limit"`
}
type Payment struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
}

type Sub struct {
	Payment
	Period int `json:"period"`
}

type Pub struct {
	Kinds kinds.T `json:"kinds"`
	Payment
}

type Fees struct {
	Admission    []Payment `json:"admission,omitempty"`
	Subscription []Sub     `json:"subscription,omitempty"`
	Publication  []Pub     `json:"publication,omitempty"`
}

type NIPs []int

type Info struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	PubKey         string  `json:"pubkey"`
	Contact        string  `json:"contact"`
	Nips           NIPs    `json:"supported_nips"`
	Software       string  `json:"software"`
	Version        string  `json:"version"`
	Limitation     *Limits `json:"limitation"`
	Retention      array.T `json:"retention"`
	RelayCountries tag.T   `json:"relay_countries"`
	LanguageTags   tag.T   `json:"language_tags"`
	Tags           tag.T   `json:"tags,omitempty"`
	PostingPolicy  string  `json:"posting_policy"`
	PaymentsURL    string  `json:"payments_url"`
	Fees           *Fees   `json:"fees,omitempty"`
	Icon           string  `json:"icon"`
	sync.Mutex
}

// NewInfo populates the nips map and if an Info structure is provided it is
// used and its nips map is populated if it isn't already.
func NewInfo(inf *Info) *Info {
	if inf != nil {
		inf.Lock()
		if inf.Limitation == nil {
			inf.Limitation = &Limits{}
		}
		inf.Unlock()
		return inf
	}
	return &Info{}
}

func (inf *Info) AddNIPs(n ...int) {
	inf.Lock()
	for _, number := range n {
		inf.Nips = append(inf.Nips, number)
	}
	inf.Unlock()
}

func (inf *Info) HasNIP(n int) (ok bool) {
	inf.Lock()
	for i := range inf.Nips {
		if inf.Nips[i] == n {
			ok = true
			break
		}
	}
	inf.Unlock()
	return
}
