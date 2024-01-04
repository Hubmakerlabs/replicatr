package relays

import (
	"context"
)

func MustRelayConnect(url string) *Relay {
	rl, e := RelayConnect(context.Background(), url)
	if e != nil {
		panic(e.Error())
	}
	return rl
}
