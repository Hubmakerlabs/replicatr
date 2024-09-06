package sdk

// Follow is a nostr account that a user is following
type Follow struct {
	Pubkey  string
	Relay   string
	Petname string
}
