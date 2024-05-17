package app

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
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
	var counter int
	for scanner.Scan() {
		counter++
		uc := context.Bg()
		ev := &event.T{}
		b := scanner.Bytes()
		if err = json.Unmarshal(b, ev); chk.E(err) {
			continue
		}
		log.I.Ln(counter, ev.ToObject().String())
		if err = upRelay.Publish(uc, ev); chk.E(err) {
			// todo: this isn't working when there is an error of invalid unclosed quotes in events
			// log.D.Ln(upAuthed)
			if strings.Contains(err.Error(), "connection closed") {
				if upRelay, err = client.Connect(c,
					args.UploadRelay); chk.E(err) {
					return 1
				}
			}
			// if !upAuthed {
			// 	log.I.Ln("authing")
			// 	// this can fail once
			// 	select {
			// 	case <-upRelay.AuthRequired:
			// 		log.T.Ln("authing to up relay")
			// 		if err = upRelay.IsAuthed(c,
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
	}
	return 0
}
