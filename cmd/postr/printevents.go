package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"mleku.dev/git/nostr/bech32encoding"
	"mleku.dev/git/nostr/event"
	"github.com/gookit/color"
)

const NostrProtocol = "nostr:"

// PrintEvents is
func (cfg *C) PrintEvents(evs []*event.T, f Follows, asJson, extra bool) {
	if asJson {
		if extra {
			var events []Event
			for _, ev := range evs {
				if profile, ok := f[ev.PubKey]; ok {
					events = append(events, Event{
						Event:   ev,
						Profile: profile,
					})
				}
			}
			for _, ev := range events {
				chk.D(json.NewEncoder(os.Stdout).Encode(ev))
			}
		} else {
			for _, ev := range evs {
				chk.D(json.NewEncoder(os.Stdout).Encode(ev))
			}
		}
		return
	}

	buf := make([]byte, 4096)
	buffer := bytes.NewBuffer(buf)
	fgRed := color.New(color.FgRed)
	fgBlue := color.New(color.Blue)
	for _, ev := range evs {
		if profile, ok := f[ev.PubKey]; ok {
			fmt.Fprintln(buffer, fgRed.Sprint(profile.Name))
			fmt.Fprintln(buffer, ev.Content)
			var rls []string
			if rls, ok = cfg.FollowsRelays[ev.PubKey]; ok {
				if nevent, err := bech32encoding.EncodeEvent(ev.ID, rls, ev.PubKey); !chk.D(err) {
					fmt.Fprint(buffer, fgBlue.Sprint(cfg.EventURLPrefix, nevent))
				}
			} else {
				note, err := bech32encoding.EncodeNote(ev.ID.String())
				if err != nil {
					note = ev.ID.String()
				}
				fmt.Fprint(buffer, fgBlue.Sprint(note))
			}
			fmt.Fprintln(buffer, " ", fgBlue.Sprint(ev.CreatedAt.Time()))
		} else {
			fmt.Fprint(buffer, fgRed.Sprint("pubkey "))
			fmt.Fprint(buffer, fgRed.Sprint(ev.PubKey))
			// fgHiBlue.Fprint(buffer, " note ID: ")
			note, err := bech32encoding.EncodeNote(ev.ID.String())
			if err != nil {
				note = ev.ID.String()
			}
			fmt.Fprint(buffer, " ", fgBlue.Sprint(cfg.EventURLPrefix, note))
			fmt.Fprint(buffer, " ", fgBlue.Sprint(ev.CreatedAt.Time()))
			fmt.Fprintln(buffer)
			fmt.Fprintln(buffer, ev.Content)
		}
		fmt.Fprintln(buffer)
	}
	fmt.Print(buffer.String())
}
