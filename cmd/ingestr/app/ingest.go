package app

import (
	"fmt"
	"time"

	"github.com/Hubmakerlabs/replicatr/pkg/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filter"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/filters"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/relay"
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
	var downRelay, upRelay *relay.T
	if downRelay, err = relay.Connect(c, args.DownloadRelay); log.E.Chk(err) {
		return 1
	}
	if upRelay, err = relay.Connect(c, args.UploadRelay); log.E.Chk(err) {
		return 1
	}
	_, _ = downRelay, upRelay
	var count int
	oldest := args.Since
	now := time.Now().Unix()
	var downAuthed, upAuthed bool
	var increment int64 = 60 * 60
	for i := oldest; i < now; i += increment {
		// create the subscription to the download relay
		var sub *subscription.T
		since := timestamp.FromUnix(i).Ptr()
		until := timestamp.FromUnix(i + increment - 1).Ptr()
		f := filters.T{
			{
				Kinds:   args.Kinds,
				Authors: tag.T{args.PubkeyHex},
				Limit:   args.Limit,
				Since:   since,
				Until:   until,
			},
			{
				Kinds: args.Kinds,
				Tags:  filter.TagMap{"#p": {args.PubkeyHex}},
				Limit: args.Limit,
				Since: since,
				Until: until,
			},
		}
		if sub, err = downRelay.Subscribe(c, f); log.Fail(err) {
			// this could fail
		}
		if !downAuthed {
			select {
			case <-downRelay.AuthRequired:
				if err = downRelay.Auth(c, func(evt *event.T) error { return evt.Sign(args.SeckeyHex) }); log.Fail(err) {
					return 1
				}
				downAuthed = true
			case <-time.After(2 * time.Second):
			}
			if sub, err = downRelay.Subscribe(c, f,
				subscription.WithLabel(fmt.Sprint(time.Now().Unix()))); log.Fail(err) {
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
				if err = upRelay.Publish(uc, ev); log.Fail(err) {
					if !upAuthed {
						// this can fail once
						select {
						case <-upRelay.AuthRequired:
							if err = upRelay.Auth(c,
								func(evt *event.T) error {
									return evt.Sign(args.SeckeyHex)
								}); log.Fail(err) {
								return 1
							}
							upAuthed = true
							if err = upRelay.Publish(uc, ev); log.Fail(err) {
								return 1
							}
						case <-time.After(2 * time.Second):
							log.E.Ln("timed out waiting to auth")
							return 1
						}
						log.I.Ln("authed")
						return 0
					} else {
						if err = upRelay.Publish(uc, ev); log.Fail(err) {
							return 1
						}
					}
				}
			}
		}
	}
	log.I.Ln("ingested", count, "events from", args.DownloadRelay, "and sent to", args.UploadRelay)
	return 0
}