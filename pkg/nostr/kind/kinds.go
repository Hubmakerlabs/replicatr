package kind

// T - which will be externally referenced as kind.T is the event type in the
// nostr protocol, the use of the capital T signifying type, consistent with Go
// idiom, the Go standard library, and much, conformant, existing code.
type T uint16

func (ki T) ToInt() int       { return int(ki) }
func (ki T) ToUint16() uint16 { return uint16(ki) }

// The event kinds are put in a separate package so they will be referred to as
// `kind.EventType` rather than `nostr.KindEventType` as this is correct Go
// idiom and the version in https://github.com/Hubmakerlabs/replicatr/pkg/nostr is unclear and
// excessive in length, impeding readability. Repeating 'nostr' in these
// constant names is redundant as they are only used in this context, and
// creating a special type for them makes this implicit and enforced by the
// compiler at compile time.
const (
	// ProfileMetadata is an event type that stores user profile data, pet
	// names, bio, lightning address, etc.
	ProfileMetadata T = 0
	// SetMetadata is a synonym for ProfileMetadata.
	SetMetadata T = 0
	// TextNote is a standard short text note of plain text a la twitter
	TextNote T = 1
	// RecommendServer is an event type that...
	RecommendServer T = 2
	RecommendRelay  T = 2
	// FollowList an event containing a list of pubkeys of users that should be
	// shown as follows in a timeline.
	FollowList T = 3
	Follows    T = 3
	// EncryptedDirectMessage is an event type that...
	EncryptedDirectMessage T = 4
	// Deletion is an event type that...
	Deletion      T = 5
	EventDeletion T = 5
	// Repost is an event type that...
	Repost T = 6
	// Reaction is an event type that...
	Reaction T = 7
	// BadgeAward is an event type
	BadgeAward T = 8
	// ReadReceipt is a type of event that marks a list of tagged events (e
	// tags) as being seen by the client, its distinctive feature is the
	// "expiration" tag which indicates a time after which the marking expires
	ReadReceipt T = 15
	// GenericRepost is an event type that...
	GenericRepost T = 16
	// ChannelCreation is an event type that...
	ChannelCreation T = 40
	// ChannelMetadata is an event type that...
	ChannelMetadata T = 41
	// ChannelMessage is an event type that...
	ChannelMessage T = 42
	// ChannelHideMessage is an event type that...
	ChannelHideMessage T = 43
	// ChannelMuteUser is an event type that...
	ChannelMuteUser T = 44
	// Bid is an event type that...
	Bid T = 1021
	// BidConfirmation is an event type that...
	BidConfirmation T = 1022
	// OpenTimestamps is an event type that...
	OpenTimestamps T = 1040
	// FileMetadata is an event type that...
	FileMetadata T = 1063
	// LiveChatMessage is an event type that...
	LiveChatMessage T = 1311
	// ProblemTracker is an event type used by Nostrocket
	ProblemTracker T = 1971
	// MemoryHole is an event type contains a report about an event (usually
	// text note or other human readable)
	MemoryHole T = 1984
	Reporting  T = 1984
	// Label is an event type that...
	Label T = 1985
	// CommunityPostApproval is an event type that...
	CommunityPostApproval T = 4550
	JobRequestStart       T = 5000
	JobRequestEnd         T = 5999
	JobResultStart        T = 6000
	JobResultEnd          T = 6999
	JobFeedback           T = 7000
	ZapGoal               T = 9041
	// ZapRequest is an event type that...
	ZapRequest T = 9734
	// Zap is an event type that...
	Zap        T = 9735
	Highlights T = 9082
	// ReplaceableStart is an event type that...
	ReplaceableStart T = 10000
	// MuteList is an event type that...
	MuteList  T = 10000
	BlockList T = 10000
	// PinList is an event type that...
	PinList T = 10001
	// RelayListMetadata is an event type that...
	RelayListMetadata     T = 10002
	BookmarkList          T = 10003
	CommunitiesList       T = 10004
	PublicChatsList       T = 10005
	BlockedRelaysList     T = 10006
	SearchRelaysList      T = 10007
	InterestsList         T = 10015
	UserEmojiList         T = 10030
	FileStorageServerList T = 10096
	// NWCWalletInfo is an event type that...
	NWCWalletInfo T = 13194
	// ReplaceableEnd is an event type that...
	ReplaceableEnd T = 20000
	// EphemeralStart is an event type that...
	EphemeralStart  T = 20000
	LightningPubRPC T = 21000
	// ClientAuthentication is an event type that...
	ClientAuthentication T = 22242
	// NWCWalletRequest is an event type that...
	NWCWalletRequest T = 23194
	WalletRequest    T = 23194
	// NWCWalletResponse is an event type that...
	NWCWalletResponse T = 23195
	WalletResponse    T = 23195
	// NostrConnect is an event type that...
	NostrConnect T = 24133
	HTTPAuth     T = 27235
	// EphemeralEnd is an event type that...
	EphemeralEnd T = 30000
	// ParameterizedReplaceableStart is an event type that...
	ParameterizedReplaceableStart T = 30000
	// CategorizedPeopleList is an event type that...
	CategorizedPeopleList T = 30000
	FollowSets            T = 30000
	// CategorizedBookmarksList is an event type that...
	CategorizedBookmarksList T = 30001
	GenericLists             T = 30001
	RelaySets                T = 30002
	BookmarkSets             T = 30003
	CurationSets             T = 30004
	// ProfileBadges is an event type that...
	ProfileBadges T = 30008
	// BadgeDefinition is an event type that...
	BadgeDefinition T = 30009
	InterestSets    T = 30015
	// StallDefinition creates or updates a stall
	StallDefinition T = 30017
	// ProductDefinition creates or updates a product
	ProductDefinition    T = 30018
	MarketplaceUIUX      T = 30019
	ProductSoldAsAuction T = 30020
	// Article is an event type that...
	Article              T = 30023
	LongFormContent      T = 30023
	DraftLongFormContent T = 30024
	EmojiSets            T = 30030
	// ApplicationSpecificData is an event type that...
	ApplicationSpecificData T = 30078

	// ParameterizedReplaceableEnd is an event type that...
	ParameterizedReplaceableEnd T = 40000
	LiveEvent                   T = 30311
	UserStatuses                T = 30315
	ClassifiedListing           T = 30402
	DraftClassifiedListing      T = 30403
	DateBasedCalendarEvent      T = 31922
	TimeBasedCalendarEvent      T = 31923
	Calendar                    T = 31924
	CalendarEventRSVP           T = 31925
	HandlerRecommendation       T = 31989
	HandlerInformation          T = 31990
	CommunityDefinition         T = 34550
)

var Map = map[T]string{
	ProfileMetadata:             "ProfileMetadata",
	TextNote:                    "TextNote",
	RecommendRelay:              "RecommendRelay",
	FollowList:                  "FollowList",
	EncryptedDirectMessage:      "EncryptedDirectMessage",
	EventDeletion:               "EventDeletion",
	Repost:                      "Repost",
	Reaction:                    "Reaction",
	BadgeAward:                  "BadgeAward",
	GenericRepost:               "GenericRepost",
	ChannelCreation:             "ChannelCreation",
	ChannelMetadata:             "ChannelMetadata",
	ChannelMessage:              "ChannelMessage",
	ChannelHideMessage:          "ChannelHideMessage",
	ChannelMuteUser:             "ChannelMuteUser",
	Bid:                         "Bid",
	BidConfirmation:             "BidConfirmation",
	OpenTimestamps:              "OpenTimestamps",
	FileMetadata:                "FileMetadata",
	LiveChatMessage:             "LiveChatMessage",
	ProblemTracker:              "ProblemTracker",
	Reporting:                   "Reporting",
	Label:                       "Label",
	CommunityPostApproval:       "CommunityPostApproval",
	JobRequestStart:             "JobRequestStart",
	JobRequestEnd:               "JobRequestEnd",
	JobResultStart:              "JobResultStart",
	JobResultEnd:                "JobResultEnd",
	JobFeedback:                 "JobFeedback",
	ZapGoal:                     "ZapGoal",
	ZapRequest:                  "ZapRequest",
	Zap:                         "Zap",
	Highlights:                  "Highlights",
	BlockList:                   "BlockList",
	PinList:                     "PinList",
	RelayListMetadata:           "RelayListMetadata",
	BookmarkList:                "BookmarkList",
	CommunitiesList:             "CommunitiesList",
	PublicChatsList:             "PublicChatsList",
	BlockedRelaysList:           "BlockedRelaysList",
	SearchRelaysList:            "SearchRelaysList",
	InterestsList:               "InterestsList",
	UserEmojiList:               "UserEmojiList",
	FileStorageServerList:       "FileStorageServerList",
	NWCWalletInfo:               "NWCWalletInfo",
	LightningPubRPC:             "LightningPubRPC",
	ClientAuthentication:        "ClientAuthentication",
	WalletRequest:               "WalletRequest",
	WalletResponse:              "WalletResponse",
	NostrConnect:                "NostrConnect",
	HTTPAuth:                    "HTTPAuth",
	FollowSets:                  "FollowSets",
	GenericLists:                "GenericLists",
	RelaySets:                   "RelaySets",
	BookmarkSets:                "BookmarkSets",
	CurationSets:                "CurationSets",
	ProfileBadges:               "ProfileBadges",
	BadgeDefinition:             "BadgeDefinition",
	InterestSets:                "InterestSets",
	StallDefinition:             "StallDefinition",
	ProductDefinition:           "ProductDefinition",
	MarketplaceUIUX:             "MarketplaceUIUX",
	ProductSoldAsAuction:        "ProductSoldAsAuction",
	LongFormContent:             "LongFormContent",
	DraftLongFormContent:        "DraftLongFormContent",
	EmojiSets:                   "EmojiSets",
	ApplicationSpecificData:     "ApplicationSpecificData",
	ParameterizedReplaceableEnd: "ParameterizedReplaceableEnd",
	LiveEvent:                   "LiveEvent",
	UserStatuses:                "UserStatuses",
	ClassifiedListing:           "ClassifiedListing",
	DraftClassifiedListing:      "DraftClassifiedListing",
	DateBasedCalendarEvent:      "DateBasedCalendarEvent",
	TimeBasedCalendarEvent:      "TimeBasedCalendarEvent",
	Calendar:                    "Calendar",
	CalendarEventRSVP:           "CalendarEventRSVP",
	HandlerRecommendation:       "HandlerRecommendation",
	HandlerInformation:          "HandlerInformation",
	CommunityDefinition:         "CommunityDefinition",
}

func (ki T) IsReplaceable() bool {
	return ki == ProfileMetadata || ki == FollowList ||
		(ki >= ReplaceableStart && ki < ReplaceableEnd)
}

func (ki T) IsEphemeral() bool {
	return ki >= EphemeralStart && ki < EphemeralEnd
}

func (ki T) IsParameterizedReplaceable() bool {
	return ki >= ParameterizedReplaceableStart &&
		ki < ParameterizedReplaceableEnd
}
