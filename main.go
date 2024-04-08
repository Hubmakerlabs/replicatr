package main

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/replicatr"
	"mleku.dev/git/nostr/context"
)

func main() {
	c, cancel := context.Cancel(context.Bg())
	replicatr.Main(os.Args, c, cancel)
}
