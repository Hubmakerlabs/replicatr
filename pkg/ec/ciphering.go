// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcec

import (
	secp "github.com/Hubmakerlabs/replicatr/pkg/ec/secp"
)

// GenerateSharedSecret generates a shared secret based on a secret key and a
// public key using Diffie-Hellman key exchange (ECDH) (RFC 4753).
// RFC5903 Section 9 states we should only return x.
func GenerateSharedSecret(privkey *SecretKey, pubkey *PublicKey) []byte {
	return secp.GenerateSharedSecret(privkey, pubkey)
}
