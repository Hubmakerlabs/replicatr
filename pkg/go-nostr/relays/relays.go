package relays

import (
	"context"
)

func MustRelayConnect(url string) *Relay {
	rl, err := RelayConnect(context.Background(), url)
	if err != nil {
		panic(err.Error())
	}
	return rl
}

