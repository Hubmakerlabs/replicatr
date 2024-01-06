// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcec

import (
	secp "github.com/Hubmakerlabs/replicatr/pkg/ec/secp"
)

// SecretKey wraps an ecdsa.SecretKey as a convenience mainly for signing
// things with the secret key without having to directly import the ecdsa
// package.
type SecretKey = secp.SecretKey
type PrivateKey = SecretKey

// SecKeyFromBytes returns a secret and public key for `curve' based on the
// secret key passed as an argument as a byte slice.
func SecKeyFromBytes(pk []byte) (*SecretKey, *PublicKey) {
	privKey := secp.SecKeyFromBytes(pk)

	return privKey, privKey.PubKey()
}

var PrivKeyFromBytes = SecKeyFromBytes

// NewSecretKey is a wrapper for ecdsa.GenerateKey that returns a SecretKey
// instead of the normal ecdsa.PrivateKey.
func NewSecretKey() (*SecretKey, error) {
	return secp.GenerateSecretKey()
}

var NewPrivateKey = NewSecretKey

// SecKeyFromScalar instantiates a new secret key from a scalar encoded as a
// big integer.
func SecKeyFromScalar(key *ModNScalar) *SecretKey {
	return &SecretKey{Key: *key}
}

var PrivKeyFromScalar = SecKeyFromScalar

// SecKeyBytesLen defines the length in bytes of a serialized secret key.
const SecKeyBytesLen = 32
const PrivKeyBytesLen = SecKeyBytesLen
