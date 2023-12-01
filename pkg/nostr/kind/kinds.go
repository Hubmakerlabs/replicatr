package kind

// T - which will be externally referenced as kind.T is the event type in the
// nostr protocol, the use of the capital T signifying type, consistent with Go
// idiom, the Go standard library, and much, conformant, existing code.
type T int

// The event kinds are put in a separate package so they will be referred to as
// `kind.EventType` rather than `nostr.KindEventType` as this is correct Go
// idiom and the version in https://github.com/nbd-wtf/go-nostr is unclear and
// excessive in length, impeding readability. Repeating 'nostr' in these
// constant names is redundant as they are only used in this context, and
// creating a special type for them makes this implicit and enforced by the
// compiler at compile timestamp.
const (
	ProfileMetadata          T = 0
	TextNote                 T = 1
	RecommendServer          T = 2
	ContactList              T = 3
	EncryptedDirectMessage   T = 4
	Deletion                 T = 5
	Repost                   T = 6
	Reaction                 T = 7
	ChannelCreation          T = 40
	ChannelMetadata          T = 41
	ChannelMessage           T = 42
	ChannelHideMessage       T = 43
	ChannelMuteUser          T = 44
	FileMetadata             T = 1063
	MemoryHole               T = 1984
	ZapRequest               T = 9734
	Zap                      T = 9735
	MuteList                 T = 10000
	PinList                  T = 10001
	RelayListMetadata        T = 10002
	NWCWalletInfo            T = 13194
	ClientAuthentication     T = 22242
	NWCWalletRequest         T = 23194
	NWCWalletResponse        T = 23195
	NostrConnect             T = 24133
	CategorizedPeopleList    T = 30000
	CategorizedBookmarksList T = 30001
	ProfileBadges            T = 30008
	BadgeDefinition          T = 30009
	StallDefinition          T = 30017
	ProductDefinition        T = 30018
	Article                  T = 30023
	ApplicationSpecificData  T = 30078
)
