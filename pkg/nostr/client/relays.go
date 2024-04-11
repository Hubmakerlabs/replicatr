package client

import (
	"mleku.dev/git/nostr/context"
)

func MustConnect(url string) *T {
	rl, err := Connect(context.Bg(), url)
	if err != nil {
		panic(err.Error())
	}
	return rl
}
