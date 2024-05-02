package main

import (
	"os"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/context"
	"github.com/Hubmakerlabs/replicatr/pkg/replicatr"
)

func main() {
	c, cancel := context.Cancel(context.Bg())
	replicatr.Main(os.Args, c, cancel)

}
