package main

import (
	"fmt"
	"github.com/Hubmakerlabs/replicatr/pkg/relay"
	"net/http"

	"github.com/Hubmakerlabs/replicatr/pkg/relay/eventstore/badger"
)

func main() {
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
