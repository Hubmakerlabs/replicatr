package main

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
)

func getPubFromSec(sk string) (pubHex string, secHex string, err error) {
	var s any
	if _, s, err = bech32encoding.Decode(sk); log.Fail(err) {
		return
	}
	secHex = s.(string)
	if pubHex, err = keys.GetPublicKey(secHex); log.Fail(err) {
		return
	}
	return
}
