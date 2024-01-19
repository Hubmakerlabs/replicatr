package kind

// T - which will be externally referenced as kind.T is the event type in the
// nostr protocol, the use of the capital T signifying type, consistent with Go
// idiom, the Go standard library, and much, conformant, existing code.
type T uint16

// The event kinds are put in a separate package so they will be referred to as
// `kind.EventType` rather than `nostr.KindEventType` as this is correct Go
// idiom and the version in https://github.com/Hubmakerlabs/replicatr/pkg/go-nostr is unclear and
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
	// FollowList an event containing a list of pubkeys of users that should be
	// shown as follows in a timeline.
	FollowList T = 3
	// EncryptedDirectMessage is an event type that...
	EncryptedDirectMessage T = 4
	// Deletion is an event type that...
	Deletion T = 5
	// Repost is an event type that...
	Repost T = 6
	// Reaction is an event type that...
	Reaction T = 7
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
	// FileMetadata is an event type that...
	FileMetadata T = 1063
	// MemoryHole is an event type that...
	MemoryHole T = 1984
	// ZapRequest is an event type that...
	ZapRequest T = 9734
	// Zap is an event type that...
	Zap T = 9735
	// ReplaceableStart is an event type that...
	ReplaceableStart T = 10000
	// MuteList is an event type that...
	MuteList T = 10000
	// PinList is an event type that...
	PinList T = 10001
	// RelayListMetadata is an event type that...
	RelayListMetadata T = 10002
	// NWCWalletInfo is an event type that...
	NWCWalletInfo T = 13194
	// ReplaceableEnd is an event type that...
	ReplaceableEnd T = 20000
	// EphemeralStart is an event type that...
	EphemeralStart T = 20000
	// ClientAuthentication is an event type that...
	ClientAuthentication T = 22242
	// NWCWalletRequest is an event type that...
	NWCWalletRequest T = 23194
	// NWCWalletResponse is an event type that...
	NWCWalletResponse T = 23195
	// NostrConnect is an event type that...
	NostrConnect T = 24133
	// EphemeralEnd is an event type that...
	EphemeralEnd T = 30000
	// ParameterizedReplaceableStart is an event type that...
	ParameterizedReplaceableStart T = 30000
	// CategorizedPeopleList is an event type that...
	CategorizedPeopleList T = 30000
	// CategorizedBookmarksList is an event type that...
	CategorizedBookmarksList T = 30001
	// ProfileBadges is an event type that...
	ProfileBadges T = 30008
	// BadgeDefinition is an event type that...
	BadgeDefinition T = 30009
	// StallDefinition is an event type that...
	StallDefinition T = 30017
	// ProductDefinition is an event type that...
	ProductDefinition T = 30018
	// Article is an event type that...
	Article T = 30023
	// ApplicationSpecificData is an event type that...
	ApplicationSpecificData T = 30078
	// ParameterizedReplaceableEnd is an event type that...
	ParameterizedReplaceableEnd T = 40000
)

func (evt T) IsReplaceable() bool {
	return evt == ProfileMetadata || evt == FollowList ||
		(evt >= ReplaceableStart && evt < ReplaceableEnd)
}

func (evt T) IsEphemeral() bool {
	return evt >= EphemeralStart && evt < EphemeralEnd
}

func (evt T) IsParameterizedReplaceable() bool {
	return evt >= ParameterizedReplaceableStart &&
		evt < ParameterizedReplaceableEnd
}
