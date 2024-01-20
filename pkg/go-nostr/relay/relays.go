package relay

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
)

func MustRelayConnect(url string) *Relay {
	rl, e := RelayConnect(context.Bg(), url)
	if e != nil {
		panic(e.Error())
	}
	return rl
}
