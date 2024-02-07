package relay

import (
	"github.com/Hubmakerlabs/replicatr/pkg/context"
)

func MustConnect(url string) *T {
	rl, err := Connect(context.Bg(), url)
	if err != nil {
		panic(err.Error())
	}
	return rl
}
