package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kind"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscription"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

func Ingest(args *Config) int {
	c := context.Bg()
	var err error
	var upRelay *client.T
	var downloadRelays []*client.T
	var downloadRelay string
	for _, downloadRelay = range args.DownloadRelays {
		var dl *client.T
		if dl, err = client.Connect(c, downloadRelay); chk.E(err) {
			return 1
		}
		downloadRelays = append(downloadRelays, dl)
		log.I.Ln("connected to download relay", downloadRelay)
	}
	if upRelay, err = client.Connect(c, args.UploadRelay); chk.E(err) {
		return 1
	}
	log.I.Ln("connected to upload relay", args.UploadRelay)
	var count int
	now := time.Now().Unix()
	var downAuthed, upAuthed bool
	var increment int64 = 500 // 6 minutes
	for i := now; i < 1640305963; i -= increment {
		// create the subscription to the download relay
		var sub *subscription.T
		since := timestamp.FromUnix(i).Ptr()
		until := timestamp.FromUnix(i + increment - 1).Ptr()
		limit := 200
		f := filters.T{
			{
				Kinds: kinds.T{kind.TextNote, kind.SetMetadata},
				Limit: &limit,
				Since: since,
				Until: until,
			},
		}
		for _, downRelay := range downloadRelays {
			if !downAuthed {
				select {
				case <-downRelay.AuthRequired:
					log.T.Ln("authing to down relay")
					if err = downRelay.Auth(c,
						func(evt *event.T) error { return evt.Sign(args.SeckeyHex) }); chk.D(err) {
						return 1
					}
					downAuthed = true
				case <-time.After(2 * time.Second):
				}
				if sub, err = downRelay.Subscribe(c, f,
					subscription.WithLabel(fmt.Sprint(time.Now().Unix()))); chk.D(err) {
					return 1
				}
			}
			if sub, err = downRelay.Subscribe(c, f); chk.D(err) {
				// this could fail
			}
		}
		if sub == nil {
			log.E.Ln("subscription failed to start")
			return 1
		}
		uc := context.Bg()
	out:
		for {
			log.D.Ln("receiving event")
			select {
			case <-c.Done():
				break out
			case <-sub.EndOfStoredEvents:
				sub.Unsub()
				break out
			case ev, more := <-sub.Events:
				if !more {
					break out
				}
				if ev == nil {
					break out
				}
				count++
				if err = upRelay.Publish(uc, ev); chk.D(err) {
					log.D.Ln(upAuthed)
					if strings.Contains(err.Error(), "connection closed") {
						if upRelay, err = client.Connect(c,
							args.UploadRelay); chk.E(err) {
							return 1
						}
					}
					if !upAuthed {
						// this can fail once
						select {
						case <-upRelay.AuthRequired:
							log.T.Ln("authing to up relay")
							if err = upRelay.Auth(c,
								func(evt *event.T) error {
									return evt.Sign(args.SeckeyHex)
								}); chk.D(err) {
								return 1
							}
							upAuthed = true
							if err = upRelay.Publish(uc, ev); chk.D(err) {
								return 1
							}
						case <-time.After(5 * time.Second):
							log.E.Ln("timed out waiting to auth")
							return 1
						}
						log.I.Ln("authed")
						return 0
					}
					if err = upRelay.Publish(uc, ev); chk.D(err) {
						return 1
					}
				}
			}
		}
		time.Sleep(time.Duration(args.Pause) * time.Second)
	}
	return 0
}
