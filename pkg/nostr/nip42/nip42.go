package nip42

import (
	"net/url"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

// CreateUnsignedAuthEvent creates an event which should be sent via an "AUTH" command.
// If the authentication succeeds, the user will be authenticated as pubkey.
func CreateUnsignedAuthEvent(challenge, pubkey, relayURL string) event.T {
	return event.T{
		PubKey:    pubkey,
		CreatedAt: timestamp.Now(),
		Kind:      kind.ClientAuthentication,
		Tags:      tags.T{{"relay", relayURL}, {"challenge", challenge}},
		Content:   "",
	}
}

// helper function for ValidateAuthEvent.
func parseURL(input string) (*url.URL, error) {
	return url.Parse(
		strings.ToLower(
			strings.TrimSuffix(input, "/"),
		),
	)
}

// ValidateAuthEvent checks whether event is a valid NIP-42 event for given challenge and relayURL.
// The result of the validation is encoded in the ok bool.
func ValidateAuthEvent(evt *event.T, challenge string,
	relayURL string) (pubkey string, ok bool) {

	if evt.Kind != kind.ClientAuthentication {
		return "", false
	}
	if evt.Tags.GetFirst([]string{"challenge", challenge}) == nil {
		return "", false
	}
	expected, err := parseURL(relayURL)
	if err != nil {
		return "", false
	}
	found, err := parseURL(evt.Tags.GetFirst([]string{"relay", ""}).Value())
	if err != nil {
		return "", false
	}
	if expected.Scheme != found.Scheme ||
		expected.Host != found.Host ||
		expected.Path != found.Path {
		return "", false
	}
	now := time.Now()
	if evt.CreatedAt.Time().After(now.Add(10*time.Minute)) || evt.CreatedAt.Time().Before(now.Add(-10*time.Minute)) {
		return "", false
	}
	// save for last, as it is most expensive operation
	// no need to check returned error, since ok == true implies err == nil.
	if ok, _ := evt.CheckSignature(); !ok {
		return "", false
	}
	return evt.PubKey, true
}
