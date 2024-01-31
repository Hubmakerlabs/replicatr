package nip42

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"mleku.online/git/slog"
)

var log = slog.New(os.Stderr, "nostr/nip42")

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
	relayURL string) (pubkey string, ok bool, err error) {

	if evt.Kind != kind.ClientAuthentication {
		err = fmt.Errorf("event incorrect kind for auth: %d %s",
			evt.Kind, kind.Map[evt.Kind])
		return
	}
	if evt.Tags.GetFirst([]string{"challenge", challenge}) == nil {
		err = fmt.Errorf("challenge tag missing from auth response")
		return
	}
	var expected, found *url.URL
	if expected, err = parseURL(relayURL); log.Fail(err) {
		return
	}
	r := evt.Tags.
		GetFirst([]string{"relay", ""}).Value()
	if r == "" {
		err = fmt.Errorf("relay tag missing from auth response")
		return
	}
	if found, err = parseURL(r); log.Fail(err) {
		err = fmt.Errorf("error parsing relay url")
		return
	}
	if expected.Scheme != found.Scheme {
		err = fmt.Errorf("HTTP Scheme incorrect: expected '%s' got '%s",
			expected.Scheme, found.Scheme)
		return
	}
	if expected.Host != found.Host {
		err = fmt.Errorf("HTTP Host incorrect: expected '%s' got '%s",
			expected.Host, found.Host)
		return
	}
	if expected.Path != found.Path {
		err = fmt.Errorf("HTTP Path incorrect: expected '%s' got '%s",
			expected.Path, found.Path)
		return
	}

	now := time.Now()
	if evt.CreatedAt.Time().After(now.Add(10*time.Minute)) || evt.CreatedAt.Time().Before(now.Add(-10*time.Minute)) {
		err = fmt.Errorf("auth event more than 10 minutes before or after current time")
		return
	}
	// save for last, as it is most expensive operation
	if ok, err = evt.CheckSignature(); !ok {
		return
	}
	pubkey = evt.PubKey
	ok = true
	return
}
