package main

import (
	"encoding/hex"
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/relay"
	log2 "mleku.online/git/log"
	"net/http"

	"github.com/Hubmakerlabs/replicatr/pkg/relay/eventstore/badger"
)

var (
	log                    = log2.GetLogger()
	fails                  = log.D.Chk
	hexDecode, encodeToHex = hex.DecodeString, hex.EncodeToString
)

func main() {
	log2.SetLogLevel(log2.Trace)
	rl := relay.New()

	db := badger.Backend{Path: "/tmp/khatru-badgern-tmp"}
	if err := db.Init(); err != nil {
		panic(err)
	}

	rl.StoreEvent = append(rl.StoreEvent, db.SaveEvent)
	rl.QueryEvents = append(rl.QueryEvents, db.QueryEvents)
	rl.CountEvents = append(rl.CountEvents, db.CountEvents)
	rl.DeleteEvent = append(rl.DeleteEvent, db.DeleteEvent)

	fmt.Println("running on :3334")
	http.ListenAndServe(":3334", rl)
}
