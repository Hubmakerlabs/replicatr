package relay

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
)

func MustConnect(url string) *Relay {
	rl, e := Connect(context.Bg(), url)
	if e != nil {
		panic(e.Error())
	}
	return rl
}
