package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fmt"

	"github.com/Hubmakerlabs/replicatr/cmd/testr/app"
	"github.com/Hubmakerlabs/replicatr/pkg/config/base"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/replicatr"
	"github.com/alexflint/go-arg"
	"github.com/fasthttp/websocket"
	"github.com/fiatjaf/eventstore/badger"
	"mleku.net/slog"
)

var conf app.Config
var relayConf base.Config
var log, chk = slog.New(os.Stderr)

func main() {
	arg.MustParse(&conf)
	var err error
	ctx, cancel := context.Cancel(context.Bg())

	// Request user input for relayArgs
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter command to run relay as usual with flags and args as needed:")
	input, _ := reader.ReadString('\n')
	relayArgs := strings.Fields(input)

	// Remove the first two elements
	if len(relayArgs) > 2 {
		relayArgs = relayArgs[2:]
	}
	os.Args = relayArgs
	arg.MustParse(&relayConf)

	if conf.Wipe {
		if relayConf.EventStore == "ic" || relayConf.EventStore == "iconly" {
			app.CanisterCleanUp(relayConf.CanisterId, relayConf.CanisterAddr,
				relayConf.SecKey)
		}

		app.BadgerCleanUp()
		os.Exit(1)
	}

	var dataDirBase string
	if dataDirBase, err = os.UserHomeDir(); chk.E(err) {
		os.Exit(1)
	}
	var dataDir string
	if relayConf.Profile == "" {
		dataDir = filepath.Join(dataDirBase, "testDB")
		relayArgs = append(relayArgs, "-p", "testDB")
	} else {
		dataDir = filepath.Join(dataDirBase, relayConf.Profile)
	}
	if relayConf.LogLevel == "" {
		relayArgs = append(relayArgs, "--loglevel", "warn")
	}
	dataDir = filepath.Join(dataDir, "filterCheckerDB")
	fmt.Println(dataDir)
	go replicatr.Main(relayArgs, ctx, cancel)
	defer cancel()
	db := &badger.BadgerBackend{Path: dataDir}
	if err = db.Init(); err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	if !conf.SkipSetup && (relayConf.EventStore == "ic" || relayConf.EventStore == "iconly") {
		app.CanisterCleanUp(relayConf.CanisterId, relayConf.CanisterAddr,
			relayConf.SecKey)
	}
	// Set up WebSocket connection
	wsURL := "ws://127.0.0.1:3334"
	var c *websocket.Conn
	c, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.F.F("Failed to connect to WebSocket: %s", err)
		return
	}
	defer c.Close()
	var ids []string
	var authors []string
	if authors, ids, err = app.EventsTest(db, conf.EventAmount, conf.Seed, ctx,
		c); err != nil {
		log.F.F("event test failed: %v", err)
	}

	if err = app.FiltersTest(authors, ids, db, conf.QueryAmount,
		ctx); err != nil {
		log.F.F("filters test failed: %v", err)
	}
	if !conf.SkipSetup {
		if relayConf.EventStore == "ic" || relayConf.EventStore == "iconly" {
			app.CanisterCleanUp(relayConf.CanisterId, relayConf.CanisterAddr,
				relayConf.SecKey)
		}
		app.BadgerCleanUp()
	}
}
