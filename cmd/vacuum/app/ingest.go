package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscription"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

func Vacuum(args *Config) int {
	c := context.Bg()
	var err error
	var upRelay *client.T
	var downloadRelay string
	var dl *client.T
	if dl, err = client.Connect(c, args.DownloadRelay); chk.E(err) {
		return 1
	}
	log.I.Ln("connected to download relay", downloadRelay)
	if upRelay, err = client.Connect(c, args.UploadRelay); chk.E(err) {
		return 1
	}
	log.I.Ln("connected to upload relay", args.UploadRelay)
	var count int
	now := time.Now().Unix()
	var downAuthed, upAuthed bool
	var increment int64 = 60
	for i := now; i > 1640305963; i -= increment {
		log.I.Ln("create the subscription to the download relay")
		var sub *subscription.T
		since := timestamp.FromUnix(i).Ptr()
		until := timestamp.FromUnix(i + increment - 1).Ptr()
		limit := 500
		f := filters.T{
			{
				Limit: &limit,
				Since: since,
				Until: until,
			},
		}
		log.I.Ln("downAuthed", downAuthed)
		if !downAuthed {
			select {
			case <-dl.AuthRequired:
				log.T.Ln("authing to down relay")
				if err = dl.Auth(c,
					func(evt *event.T) error { return evt.Sign(args.SeckeyHex) }); chk.D(err) {
					return 1
				}
				downAuthed = true
			case <-time.After(2 * time.Second):
			}
			log.I.Ln("creating download subscription")
			if sub, err = dl.Subscribe(c, f,
				subscription.WithLabel(fmt.Sprint(time.Now().Unix()))); chk.D(err) {
				return 1
			}
			log.I.Ln("created download subscription")
		} else {
			if sub, err = dl.Subscribe(c, f); chk.D(err) {
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
		time.Sleep(time.Duration(args.Pause) * time.Millisecond * time.Second)
	}
	return 0
}
