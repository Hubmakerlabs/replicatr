package main

import (
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/bech32encoding"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/keys"
)

func getPubFromSec(sk string) (pubHex string, secHex string, e error) {
	var s any
	if _, s, e = bech32encoding.Decode(sk); log.Fail(e) {
		return
	}
	secHex = s.(string)
	if pubHex, e = keys.GetPublicKey(secHex); log.Fail(e) {
		return
	}
	return
}
