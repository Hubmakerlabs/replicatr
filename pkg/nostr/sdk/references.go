package sdk

import (
	"regexp"
	"strconv"
	"strings"

	"mleku.dev/git/nostr/bech32encoding"
	"mleku.dev/git/nostr/event"
	"mleku.dev/git/nostr/eventid"
	"mleku.dev/git/nostr/kind"
	"mleku.dev/git/nostr/pointers"
)

type Reference struct {
	Text    string
	Start   int
	End     int
	Profile *pointers.Profile
	Event   *pointers.Event
	Entity  *pointers.Entity
}

var mentionRegex = regexp.MustCompile(`\bnostr:((note|npub|naddr|nevent|nprofile)1\w+)\b|#\[(\d+)\]`)

// ParseReferences parses both NIP-08 and NIP-27 references in a single unifying
// interface.
func ParseReferences(evt *event.T) (refs []*Reference) {
	content := evt.Content
	for _, r := range mentionRegex.
		FindAllStringSubmatchIndex(evt.Content, -1) {
		ref := &Reference{
			Text:  content[r[0]:r[1]],
			Start: r[0],
			End:   r[1],
		}
		if r[6] == -1 {
			// didn't find a NIP-10 #[0] reference, so it's a NIP-27 mention
			nip19code := content[r[2]:r[3]]
			if prefix, data, err := bech32encoding.Decode(nip19code); !log.D.Chk(err) {
				switch prefix {
				case "npub":
					ref.Profile = &pointers.Profile{
						PublicKey: data.(string),
						Relays:    []string{},
					}
				case "nprofile":
					pp := data.(pointers.Profile)
					ref.Profile = &pp
				case "note":
					ref.Event = &pointers.Event{
						ID:     data.(eventid.T),
						Relays: []string{},
					}
				case "nevent":
					evp := data.(pointers.Event)
					ref.Event = &evp
				case "naddr":
					addr := data.(pointers.Entity)
					ref.Entity = &addr
				}
			}
		} else {
			// it's a NIP-10 mention.
			// parse the number, get data from event tags.
			n := content[r[6]:r[7]]
			idx, err := strconv.Atoi(n)
			if chk.D(err) || len(evt.Tags) <= idx {
				continue
			}
			if tag := evt.Tags[idx]; tag != nil && len(tag) >= 2 {
				switch tag[0] {
				case "p":
					relays := make([]string, 0, 1)
					if len(tag) > 2 && tag[2] != "" {
						relays = append(relays, tag[2])
					}
					ref.Profile = &pointers.Profile{
						PublicKey: tag[1],
						Relays:    relays,
					}
				case "e":
					relays := make([]string, 0, 1)
					if len(tag) > 2 && tag[2] != "" {
						relays = append(relays, tag[2])
					}
					ref.Event = &pointers.Event{
						ID:     eventid.T(tag[1]),
						Relays: relays,
					}
				case "a":
					if parts := strings.Split(tag[1], ":"); len(parts) == 3 {
						k, _ := strconv.Atoi(parts[0])
						relays := make([]string, 0, 1)
						if len(tag) > 2 && tag[2] != "" {
							relays = append(relays, tag[2])
						}
						ref.Entity = &pointers.Entity{
							Identifier: parts[2],
							PublicKey:  parts[1],
							Kind:       kind.T(k),
							Relays:     relays,
						}
					}
				}
			}
		}
		refs = append(refs, ref)
	}
	return
}
