package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/Hubmakerlabs/replicatr/khatru"
	"mleku.dev/git/nostr/eventstore/badger"
)

func main() {
	relay := khatru.NewRelay()

	db := badger.Backend{
		Path: "/tmp/khatru-badgern-tmp",
		WG:   &sync.WaitGroup{},
	}
	if err := db.Init(); err != nil {
		panic(err)
	}

	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)

	fmt.Println("running on :3334")
	http.ListenAndServe(":3334", relay)
}
