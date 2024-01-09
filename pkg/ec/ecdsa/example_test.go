// Copyright (c) 2014 The btcsuite developers
// Copyright (c) 2015-2021 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// TODO: change this to work with sha256

package ecdsa_test

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/ec/ecdsa"
	"github.com/Hubmakerlabs/replicatr/pkg/ec/secp"
	"github.com/Hubmakerlabs/replicatr/pkg/hex"
	"github.com/dchest/blake256"
)

// This example demonstrates signing a message with a secp256k1 secret key that
// is first parsed from raw bytes and serializing the generated signature.
func ExampleSign() {
	// Decode a hex-encoded secret key.
	pkBytes, e := hex.Dec("22a47fa09a223f2aa079edf85a7c2d4f87" +
		"20ee63e502ee2869afab7de234b80c")
	if e != nil {
		fmt.Println(e)
		return
	}
	secKey := secp256k1.SecKeyFromBytes(pkBytes)

	// Sign a message using the secret key.
	message := "test message"
	messageHash := blake256.Sum256([]byte(message))
	signature := ecdsa.Sign(secKey, messageHash[:])

	// Serialize and display the signature.
	fmt.Printf("Serialized Signature: %x\n", signature.Serialize())

	// Verify the signature for the message using the public key.
	pubKey := secKey.PubKey()
	verified := signature.Verify(messageHash[:], pubKey)
	fmt.Printf("Signature Verified? %v\n", verified)

	// Output:
	// Serialized Signature: 3045022100fcc0a8768cfbcefcf2cadd7cfb0fb18ed08dd2e2ae84bef1a474a3d351b26f0302200fc1a350b45f46fa00101391302818d748c2b22615511a3ffd5bb638bd777207
	// Signature Verified? true
}

// This example demonstrates verifying a secp256k1 signature against a public
// key that is first parsed from raw bytes.  The signature is also parsed from
// raw bytes.
func ExampleSignature_Verify() {
	// Decode hex-encoded serialized public key.
	pubKeyBytes, e := hex.Dec("02a673638cb9587cb68ea08dbef685c" +
		"6f2d2a751a8b3c6f2a7e9a4999e6e4bfaf5")
	if e != nil {
		fmt.Println(e)
		return
	}
	pubKey, e := secp256k1.ParsePubKey(pubKeyBytes)
	if e != nil {
		fmt.Println(e)
		return
	}

	// Decode hex-encoded serialized signature.
	sigBytes, e := hex.Dec("3045022100fcc0a8768cfbcefcf2cadd7cfb0" +
		"fb18ed08dd2e2ae84bef1a474a3d351b26f0302200fc1a350b45f46fa0010139130" +
		"2818d748c2b22615511a3ffd5bb638bd777207")
	if e != nil {
		fmt.Println(e)
		return
	}
	signature, e := ecdsa.ParseDERSignature(sigBytes)
	if e != nil {
		fmt.Println(e)
		return
	}

	// Verify the signature for the message using the public key.
	message := "test message"
	messageHash := blake256.Sum256([]byte(message))
	verified := signature.Verify(messageHash[:], pubKey)
	fmt.Println("Signature Verified?", verified)

	// Output:
	// Signature Verified? true
}
