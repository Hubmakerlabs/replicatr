package app

import (
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/envelopes/eventenvelope"
	"github.com/Hubmakerlabs/replicatr/pkg/units"
)

func Blower(args *Config) int {
	c := context.Bg()
	var err error
	var upRelay *client.T
	if upRelay, err = client.Connect(c, args.UploadRelay); chk.E(err) {
		return 1
	}
	log.I.Ln("connected to upload relay", args.UploadRelay)
	// var upAuthed bool
	var fh *os.File
	if fh, err = os.OpenFile(args.SourceFile, os.O_RDONLY, 0755); chk.D(err) {
		return 1
	}
	buf := make([]byte, app.MaxMessageSize)
	scanner := bufio.NewScanner(fh)
	scanner.Buffer(buf, 500000000)
	var counter, position int
	for scanner.Scan() {
		counter++
		b := scanner.Bytes()
		position += len(b)
		if counter <= args.Skip {
			continue
		}
		if len(b) > app.MaxMessageSize {
			log.I.Ln("message too long", string(b))
			continue
		}
		log.I.F("%d %0.6f Gb", counter, float64(position)/float64(units.Gb))
		if err = <-upRelay.Write(eventenvelope.FromRawJSON("", b)); chk.E(err) {
			if strings.Contains(err.Error(), "connection closed") {
				if upRelay, err = client.Connect(c,
					args.UploadRelay); chk.E(err) {
					return 1
				}
			}
			// todo: get authing working properly
			// if !upAuthed {
			// 	log.I.Ln("authing")
			// 	// this can fail once
			// 	select {
			// 	case <-upRelay.AuthRequired:
			// 		log.T.Ln("authing to up relay")
			// 		if err = upRelay.Auth(c,
			// 			func(evt *event.T) error {
			// 				return evt.Sign(args.SeckeyHex)
			// 			}); chk.D(err) {
			// 			return 1
			// 		}
			// 		upAuthed = true
			// 		if err = upRelay.Publish(uc, ev); chk.D(err) {
			// 			return 1
			// 		}
			// 	case <-time.After(5 * time.Second):
			// 		log.E.Ln("timed out waiting to auth")
			// 		return 1
			// 	}
			// 	log.I.Ln("authed")
			// 	return 0
			// }
			// if err = upRelay.Publish(uc, ev); chk.D(err) {
			// 	return 1
			// }
		}
		time.Sleep(time.Millisecond * 20)
	}
	return 0
}
