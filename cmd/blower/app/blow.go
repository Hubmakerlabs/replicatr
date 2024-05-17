package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Hubmakerlabs/replicatr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/client"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/event"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/hex"
	"github.com/Hubmakerlabs/replicatr/pkg/units"
	"github.com/minio/sha256-simd"
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
	var fi os.FileInfo
	if fi, err = fh.Stat(); chk.E(err) {
		return 1
	}
	totalSize := fi.Size()
	buf := make([]byte, 0, app.MaxMessageSize)
	scanner := bufio.NewScanner(fh)
	scanner.Buffer(buf, 500000000)
	var counter, position int
	var start int64
	var u *client.T
	if u, err = client.ConnectWithAuth(c, args.UploadRelay, args.SeckeyHex); chk.E(err) {
		os.Exit(1)
	}
	defer u.ConnectionContextCancel()
	for scanner.Scan() {
		counter++
		b := scanner.Bytes()
		position += len(b)
		if counter <= args.Skip {
			continue
		}
		if start == 0 {
			start = time.Now().Unix()
		}
		rb := make([]byte, len(b))
		copy(rb, b)
		ev := &event.T{}
		if err = json.Unmarshal(b, ev); chk.E(err) {
			continue
		}
		if string(b) != ev.ToObject().String() {
			fmt.Printf("invalid JSON:\n%s\n%s\n", string(b), ev.ToObject().String())
			continue
		}
		can := ev.ToCanonical().Bytes()
		id := sha256.Sum256(can)
		idh := hex.Enc(id[:])
		if idh != string(ev.ID) {
			log.W.Ln("mismatch between original and encoded/decoded", hex.Enc(id[:]), string(ev.ID))
			continue
		}
	retry:
		for {
			// retry := time.After(time.Second)
			// ch := u.Write(eventenvelope.FromRawJSON("", b))
			err = u.Publish(c, ev)
			// select {
			// case err = <-ch:
			if err == nil {
				break retry
			}
			log.E.Ln(err.Error())
			if strings.Contains(err.Error(), "failed to flush writer") {
				return 1
			}
			if strings.Contains(err.Error(), "connection closed") {
				// upRelay.Close()
				// upRelay.Connection.Conn.Close()
				upRelay.ConnectionContextCancel()
				if upRelay, err = client.ConnectWithAuth(c,
					args.UploadRelay, args.SeckeyHex); chk.E(err) {
					break retry
				}
			}
			// if err != nil {
			// 	break retry
			// }
			// case <-retry:
			// 	u.ConnectionContextCancel()
			// 	log.I.Ln("reconnecting")
			// 	if u, err = client.Connect(c, args.UploadRelay); chk.E(err) {
			// 		return 1
			// 	}
			// }
			// }
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
		log.I.F("%d %6d bytes %0.6fGb %0.3f done %s",
			counter, len(b), float64(position)/float64(units.Gb),
			float64(position)/float64(totalSize),
			ev.ID.String(),
		)
		// }(rb, counter)
		// time.Sleep(time.Second / 20)
	}
	return 0
}
