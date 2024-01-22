package main

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/fatih/color"
)

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
				log.Fail(json.NewEncoder(os.Stdout).Encode(ev))
			}
		} else {
			for _, ev := range evs {
				log.Fail(json.NewEncoder(os.Stdout).Encode(ev))
			}
		}
		return
	}

	buf := make([]byte, 4096)
	buffer := bytes.NewBuffer(buf)
	fgHiRed := color.New(color.FgHiRed, color.Bold)
	fgRed := color.New(color.FgRed)
	fgNormal := color.New(color.Reset)
	fgHiBlue := color.Set(color.FgHiBlue)
	for _, ev := range evs {
		profile, ok := f[ev.PubKey]
		if ok {
			color.Set(color.FgHiRed)
			fgHiRed.Fprintln(buffer, profile.Name)
			fgNormal.Fprintln(buffer, ev.Content)
			note, e := bech32encoding.EncodeNote(ev.ID.String())
			if e != nil {
				note = ev.ID.String()
			}
			fgHiBlue.Fprint(buffer, note)
			fgHiBlue.Fprintln(buffer, " ", ev.CreatedAt.Time())
		} else {
			fgRed.Fprint(buffer, "pubkey ")
			fgRed.Fprint(buffer, ev.PubKey)
			// fgHiBlue.Fprint(buffer, " note ID: ")
			note, e := bech32encoding.EncodeNote(ev.ID.String())
			if e != nil {
				note = ev.ID.String()
			}
			fgHiBlue.Fprint(buffer, " ", note)
			fgHiBlue.Fprint(buffer, " ", ev.CreatedAt.Time())
			fgHiBlue.Fprintln(buffer)
			fgNormal.Fprintln(buffer, ev.Content)
		}
		fgNormal.Fprintln(buffer)
	}
	fgNormal.Print(buffer.String())
}
