package auth

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tags"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

const Required = "auth-required"

// CreateUnsigned creates an event which should be sent via an "AUTH" command.
// If the authentication succeeds, the user will be authenticated as pubkey.
func CreateUnsigned(challenge, relayURL string) *event.T {
	return &event.T{
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

// Validate checks whether event is a valid NIP-42 event for given challenge and relayURL.
// The result of the validation is encoded in the ok bool.
func Validate(evt *event.T, challenge string,
	relayURL string) (pubkey string, ok bool, err error) {

	if evt.Kind != kind.ClientAuthentication {
		err = log.E.Err("event incorrect kind for auth: %d %s",
			evt.Kind, kind.Map[evt.Kind])
		log.D.Ln(err)
		return
	}
	if evt.Tags.GetFirst([]string{"challenge", challenge}) == nil {
		err = log.E.Err("challenge tag missing from auth response")
		log.D.Ln(err)
		return
	}
	var expected, found *url.URL
	if expected, err = parseURL(relayURL); chk.D(err) {
		log.D.Ln(err)
		return
	}
	r := evt.Tags.
		GetFirst([]string{"relay", ""}).Value()
	if r == "" {
		err = log.E.Err("relay tag missing from auth response")
		log.D.Ln(err)
		return
	}
	if found, err = parseURL(r); chk.D(err) {
		err = log.E.Err("error parsing relay url: %s", err)
		log.D.Ln(err)
		return
	}
	if expected.Scheme != found.Scheme {
		err = log.E.Err("HTTP Scheme incorrect: expected '%s' got '%s",
			expected.Scheme, found.Scheme)
		log.D.Ln(err)
		return
	}
	if expected.Host != found.Host {
		err = log.E.Err("HTTP Host incorrect: expected '%s' got '%s",
			expected.Host, found.Host)
		log.D.Ln(err)
		return
	}
	if expected.Path != found.Path {
		err = log.E.Err("HTTP Path incorrect: expected '%s' got '%s",
			expected.Path, found.Path)
		log.D.Ln(err)
		return
	}
	now := time.Now()
	if evt.CreatedAt.Time().After(now.Add(10*time.Minute)) ||
		evt.CreatedAt.Time().Before(now.Add(-10*time.Minute)) {
		err = log.E.Err(
			"auth event more than 10 minutes before or after current time")
		log.D.Ln(err)
		return
	}
	// save for last, as it is most expensive operation
	if ok, err = evt.CheckSignature(); !ok {
		log.D.Ln(err)
		return
	}
	pubkey = evt.PubKey
	ok = true
	return
}
