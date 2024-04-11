package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/subscription"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

func Ingest(args *Config) int {
	// there is no legitimate event with a time earlier than fiatjaf's first
	// event so this is the boundary we set
	if args.Since == 0 {
		args.Since = 1640305963
	}
	// if no kinds are given, use the default
	if len(args.Kinds) == 0 {
		args.Kinds = defaultKinds
	}
	log.I.F("ingesting events of kind %v with dates after %v from %s and sending to %s",
		args.Kinds,
		time.Unix(args.Since, 0),
		args.DownloadRelay,
		args.UploadRelay,
	)
	c := context.Bg()
	var err error
	var downRelay, upRelay *client.T
	if downRelay, err = client.Connect(c, args.DownloadRelay); chk.E(err) {
		return 1
	}
	log.I.Ln("connected to download relay")
	if upRelay, err = client.Connect(c, args.UploadRelay); chk.E(err) {
		return 1
	}
	log.I.Ln("connected to upload relay")
	var count int
	oldest := args.Since
	now := time.Now().Unix()
	var downAuthed, upAuthed bool
	var increment = 60 * 60 * args.Interval
	for i := oldest; i < now; i += increment {
		// create the subscription to the download relay
		var sub *subscription.T
		since := timestamp.FromUnix(i).Ptr()
		until := timestamp.FromUnix(i + increment - 1).Ptr()
		authors := append(tag.T{args.PubkeyHex}, args.OtherPubkeys...)
		pTags := filter.TagMap{"#p": authors}
		f := filters.T{
			{
				Kinds:   args.Kinds,
				Authors: tag.T{args.PubkeyHex},
				Limit:   &args.Limit,
				Since:   since,
				Until:   until,
			},
			{
				Kinds: args.Kinds,
				Tags:  pTags,
				Limit: &args.Limit,
				Since: since,
				Until: until,
			},
		}
		if sub, err = downRelay.Subscribe(c, f); chk.D(err) {
			// this could fail
		}
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
	log.I.Ln("ingested", count, "events from", args.DownloadRelay,
		"and sent to", args.UploadRelay)
	return 0
}
