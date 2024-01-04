package sdk

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/nip19"
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/pointers"
)

type Reference struct {
	Text    string
	Start   int
	End     int
	Profile *pointers.ProfilePointer
	Event   *pointers.EventPointer
	Entity  *pointers.EntityPointer
}

var mentionRegex = regexp.MustCompile(`\bnostr:((note|npub|naddr|nevent|nprofile)1\w+)\b|#\[(\d+)\]`)

// ParseReferences parses both NIP-08 and NIP-27 references in a single unifying interface.
func ParseReferences(evt *event.T) []*Reference {
	var references []*Reference
	content := evt.Content

	for _, ref := range mentionRegex.FindAllStringSubmatchIndex(evt.Content, -1) {
		reference := &Reference{
			Text:  content[ref[0]:ref[1]],
			Start: ref[0],
			End:   ref[1],
		}

		if ref[6] == -1 {
			// didn't find a NIP-10 #[0] reference, so it's a NIP-27 mention
			nip19code := content[ref[2]:ref[3]]

			if prefix, data, err := nip19.Decode(nip19code); err == nil {
				switch prefix {
				case "npub":
					reference.Profile = &pointers.ProfilePointer{
						PublicKey: data.(string), Relays: []string{},
					}
				case "nprofile":
					pp := data.(pointers.ProfilePointer)
					reference.Profile = &pp
				case "note":
					reference.Event = &pointers.EventPointer{ID: data.(string), Relays: []string{}}
				case "nevent":
					evp := data.(pointers.EventPointer)
					reference.Event = &evp
				case "naddr":
					addr := data.(pointers.EntityPointer)
					reference.Entity = &addr
				}
			}
		} else {
			// it's a NIP-10 mention.
			// parse the number, get data from event tags.
			n := content[ref[6]:ref[7]]
			idx, err := strconv.Atoi(n)
			if err != nil || len(evt.Tags) <= idx {
				continue
			}
			if tag := evt.Tags[idx]; tag != nil && len(tag) >= 2 {
				switch tag[0] {
				case "p":
					relays := make([]string, 0, 1)
					if len(tag) > 2 && tag[2] != "" {
						relays = append(relays, tag[2])
					}
					reference.Profile = &pointers.ProfilePointer{
						PublicKey: tag[1],
						Relays:    relays,
					}
				case "e":
					relays := make([]string, 0, 1)
					if len(tag) > 2 && tag[2] != "" {
						relays = append(relays, tag[2])
					}
					reference.Event = &pointers.EventPointer{
						ID:     tag[1],
						Relays: relays,
					}
				case "a":
					if parts := strings.Split(tag[1], ":"); len(parts) == 3 {
						kind, _ := strconv.Atoi(parts[0])
						relays := make([]string, 0, 1)
						if len(tag) > 2 && tag[2] != "" {
							relays = append(relays, tag[2])
						}
						reference.Entity = &pointers.EntityPointer{
							Identifier: parts[2],
							PublicKey:  parts[1],
							Kind:       kind,
							Relays:     relays,
						}
					}
				}
			}
		}

		references = append(references, reference)
	}

	return references
}
