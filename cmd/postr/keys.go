package main

import (
	"github.com/Hubmakerlabs/replicatr/pkg/go-nostr/keys"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/nip19"
)

func getPubFromSec(sk string) (pubHex string, secHex string, e error) {
	var s any
	if _, s, e = nip19.Decode(sk); log.Fail(e) {
		return
	}
	secHex = s.(string)
	if pubHex, e = keys.GetPublicKey(secHex); log.Fail(e) {
		return
	}
	return
}
