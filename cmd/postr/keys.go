package main

import (
	"mleku.dev/git/nostr/bech32encoding"
	"mleku.dev/git/nostr/keys"
)

func getPubFromSec(sk string) (pubHex string, secHex string, err error) {
	var s any
	if _, s, err = bech32encoding.Decode(sk); chk.D(err) {
		return
	}
	secHex = s.(string)
	if pubHex, err = keys.GetPublicKey(secHex); chk.D(err) {
		return
	}
	return
}
