// Copyright (c) 2015-2020 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package schnorr

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/minio/sha256-simd"
	"mleku.online/git/ec/secp"
)

// TestSignatureParsing ensures that signatures are properly parsed including
// error paths.
func TestSignatureParsing(t *testing.T) {
	tests := []struct {
		name string // test description
		sig  string // hex encoded signature to parse
		err  error  // expected error
	}{{
		name: "valid signature 1",
		sig: "c6ec70969d8367538c442f8e13eb20ff0c9143690f31cd3a384da54dd29ec0aa" +
			"4b78a1b0d6b4186195d42a85614d3befd9f12ed26542d0dd1045f38c98b4a405",
		err: nil,
	}, {
		name: "valid signature 2",
		sig: "adc21db084fa1765f9372c2021fb298720f3d13e6d844e2dff751a2d46a69277" +
			"0b989e316f7faf308a5f4a7343c0569465287cf6bff457250d6dacbb361f6e63",
		err: nil,
	}, {
		name: "empty",
		sig:  "",
		err:  ErrSigTooShort,
	}, {
		name: "too short by one byte",
		sig: "adc21db084fa1765f9372c2021fb298720f3d13e6d844e2dff751a2d46a69277" +
			"0b989e316f7faf308a5f4a7343c0569465287cf6bff457250d6dacbb361f6e",
		err: ErrSigTooShort,
	}, {
		name: "too long by one byte",
		sig: "adc21db084fa1765f9372c2021fb298720f3d13e6d844e2dff751a2d46a69277" +
			"0b989e316f7faf308a5f4a7343c0569465287cf6bff457250d6dacbb361f6e6300",
		err: ErrSigTooLong,
	}, {
		name: "r == p",
		sig: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f" +
			"181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d09",
		err: ErrSigRTooBig,
	}, {
		name: "r > p",
		sig: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc30" +
			"181522ec8eca07de4860a4acdd12909d831cc56cbbac4622082221a8768d1d09",
		err: ErrSigRTooBig,
	}, {
		name: "s == n",
		sig: "4e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd41" +
			"fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141",
		err: ErrSigSTooBig,
	}, {
		name: "s > n",
		sig: "4e45e16932b8af514961a1d3a1a25fdf3f4f7732e9d624c6c61548ab5fb8cd41" +
			"fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364142",
		err: ErrSigSTooBig,
	}}

	for _, test := range tests {
		_, err := ParseSignature(hexToBytes(test.sig))
		if !errors.Is(err, test.err) {
			t.Errorf("%s mismatched err -- got %v, want %v", test.name, err,
				test.err)
			continue
		}
	}
}

// TestSchnorrSignAndVerify ensures the Schnorr signing function produces the
// expected signatures for a selected set of secret keys, messages, and nonces
// that have been independently verified with the Sage computer algebra system.
// It also ensures verifying the signature works as expected.
func TestSchnorrSignAndVerify(t *testing.T) {
	tests := []struct {
		name     string // test description
		key      string // hex encded secret key
		msg      string // hex encoded message to sign before hashing
		hash     string // hex encoded hash of the message to sign
		nonce    string // hex encoded nonce to use in the signature calculation
		rfc6979  bool   // whether the nonce is an RFC6979 nonce
		expected string // expected signature
	}{
		{
			name:     "key 0x1, sha256(0x01020304), rfc6979 nonce",
			key:      "0000000000000000000000000000000000000000000000000000000000000001",
			msg:      "01020304",
			hash:     "9f64a747e1b97f131fabb6b447296c9b6f0201e79fb3c5356e6c77e89b6a806a",
			nonce:    "475840cefb3c671398234e0703ccd06dcb5bc99e15c75b0b1d1068bfcf372fc4",
			rfc6979:  true,
			expected: "247dcfd9ec96ba988e38e7e391ee35c33f2388b9153334f350370b193375ddfd7fd297d3bc673ead07c62c1a020de411605fce8537eab6a81dd06539b44619f3",
		},
		{
			name:     "key 0x1, sha256(0x01020304), random nonce",
			key:      "0000000000000000000000000000000000000000000000000000000000000001",
			msg:      "01020304",
			hash:     "9f64a747e1b97f131fabb6b447296c9b6f0201e79fb3c5356e6c77e89b6a806a",
			nonce:    "a6df66500afeb7711d4c8e2220960855d940a5ed57260d2c98fbf6066cca283e",
			rfc6979:  false,
			expected: "b073759a96a835b09b79e7b93c37fdbe48fb82b000c4a0e1404ba5d1fbc15d0ae76fbc5c2b1b69186a57de423c2c8bb541b138405ef641263ecc1ef66ee65e34",
		},
		{
			name:     "key 0x2, sha256(0x01020304), rfc6979 nonce",
			key:      "0000000000000000000000000000000000000000000000000000000000000002",
			msg:      "01020304",
			hash:     "9f64a747e1b97f131fabb6b447296c9b6f0201e79fb3c5356e6c77e89b6a806a",
			nonce:    "ccb8217e1d3a38c8ce410a0b34b9f7705be44cbf2202cfcaa32fa80efd80eb54",
			rfc6979:  true,
			expected: "9429c5031738a552a7ba046aef38b67da56b5721c84e33ee27c642eefd4248fc39a72eb8f329fceff4221c637304268b376e6e735b689196a155a8abc1e8746c",
		},
		{
			name:     "key 0x2, sha256(0x01020304), random nonce",
			key:      "0000000000000000000000000000000000000000000000000000000000000002",
			msg:      "01020304",
			hash:     "9f64a747e1b97f131fabb6b447296c9b6f0201e79fb3c5356e6c77e89b6a806a",
			nonce:    "679a6d36e7fe6c02d7668af86d78186e8f9ccc04371ac1c8c37939d1f5cae07a",
			rfc6979:  false,
			expected: "4a090d82f48ca12d9e7aa24b5dcc187ee0db2920496f671d63e86036aaa7997e85f15ab0771b3595ba116791f33d0d4c3236f8aa5bd17ee872898125798b2792",
		},
		{
			name:     "key 0x1, sha256(0x0102030405), rfc6979 nonce",
			key:      "0000000000000000000000000000000000000000000000000000000000000001",
			msg:      "0102030405",
			hash:     "74f81fe167d99b4cb41d6d0ccda82278caee9f3e2f25d5e5a3936ff3dcec60d0",
			nonce:    "5e0f1a9d538283fb33d172e5be4000b231822cf3fa8688dbabbaf6fd3897e565",
			rfc6979:  true,
			expected: "8ddc6a0203d0c63b30454fe3cc06caf23a90667403cb4d0c5836f501810ff0aa1fa92bbdbf61d005bcb92003f3b16e82c8c36abc71aeb014f218ebaf51ca2962",
		},
		{
			name:     "key 0x1, sha256(0x0102030405), random nonce",
			key:      "0000000000000000000000000000000000000000000000000000000000000001",
			msg:      "0102030405",
			hash:     "74f81fe167d99b4cb41d6d0ccda82278caee9f3e2f25d5e5a3936ff3dcec60d0",
			nonce:    "65f880c892fdb6e7f74f76b18c7c942cfd037ef9cf97c39c36e08bbc36b41616",
			rfc6979:  false,
			expected: "72e5666f4e9d1099447b825cf737ee32112f17a67e2ca7017ae098da31dfbb8b6b2f66851da1e8d0fa66dbd2c0ba0faddf6906def53ede3a2b49825440c960e3",
		},
		{
			name:     "key 0x2, sha256(0x0102030405), rfc6979 nonce",
			key:      "0000000000000000000000000000000000000000000000000000000000000002",
			msg:      "0102030405",
			hash:     "74f81fe167d99b4cb41d6d0ccda82278caee9f3e2f25d5e5a3936ff3dcec60d0",
			nonce:    "b60cca1fdf081e14c342f9b557a345894542d618c2afefa9d1f8324d23979771",
			rfc6979:  true,
			expected: "dcb0ff92f2a2798c1229e82e1397cc2af63845550b82818e23b8e7b2ff409ed2e201e1d7d8718eba5ab87ae34b05df14a4d15515996912de1bd8d577e378fb93",
		},
		{
			name:     "key 0x2, sha256(0x0102030405), random nonce",
			key:      "0000000000000000000000000000000000000000000000000000000000000002",
			msg:      "0102030405",
			hash:     "74f81fe167d99b4cb41d6d0ccda82278caee9f3e2f25d5e5a3936ff3dcec60d0",
			nonce:    "026ece4cfb704733dd5eef7898e44c33bd5a0d749eb043f48705e40fa9e9afa0",
			rfc6979:  false,
			expected: "3c4c5a2f217ea758113fd4e89eb756314dfad101a300f48e5bd764d3b6e0f8bf7b8f126dccde7a581318e000d7da3791ebaee51a249c1f31091fb037c3b33b15",
		},
		{
			name:     "random key 1, sha256(0x01), rfc6979 nonce",
			key:      "a1becef2069444a9dc6331c3247e113c3ee142edda683db8643f9cb0af7cbe33",
			msg:      "01",
			hash:     "4bf5122f344554c53bde2ebb8cd2b7e3d1600ad631c385a5d7cce23c7785459a",
			nonce:    "a5659d3f030cbae8959ebdfed6d4e379351a8a373053f25edc988e7233f18740",
			rfc6979:  true,
			expected: "1a6f2a06a413d7695ba885b4e95e3afee7d548e1855678e4a8b4aa63ec326b30f37e073f683d4d439ddcf94dd612e326c4a887d5b654e64429f4d296d690c49b",
		},
		{
			name:     "random key 2, sha256(0x02), rfc6979 nonce",
			key:      "59930b76d4b15767ec0e8c8e5812aa2e57db30c6af7963e2a6295ba02af5416b",
			msg:      "02",
			hash:     "dbc1b4c900ffe48d575b5da5c638040125f65db0fe3e24494b76ea986457d986",
			nonce:    "b6e1a09335e8ed6c4ec1c5983c6adc475c3b64076654170933c7f8556de26047",
			rfc6979:  true,
			expected: "6d0221f6f3132fc2c7366c842e0b67de6b82ada47c2542937718c200d0b8c48a49996537e7936f536c5b345a5b85c5352278f178ab38549718b30f42288f220f",
		},
		{
			name:     "random key 3, sha256(0x03), rfc6979 nonce",
			key:      "c5b205c36bb7497d242e96ec19a2a4f086d8daa919135cf490d2b7c0230f0e91",
			msg:      "03",
			hash:     "084fed08b978af4d7d196a7446a86b58009e636b611db16211b65a9aadff29c5",
			nonce:    "019496ff728dcaad13bfb281fd63637840c2cc45813a3d730e502ab0ce5f1480",
			rfc6979:  true,
			expected: "6d882b5e507b6b0215c10e9ce1e53a51205446bce505bfede92d42332ccc95f5b4965837641d4bfd521f01ebfd5a6351b5b409695dc656a5f4584de242417471",
		},
		{
			name:     "random key 4, sha256(0x04), rfc6979 nonce",
			key:      "65b46d4eb001c649a86309286aaf94b18386effe62c2e1586d9b1898ccf0099b",
			msg:      "04",
			hash:     "e52d9c508c502347344d8c07ad91cbd6068afc75ff6292f062a09ca381c89e71",
			nonce:    "b32a5cf8f11f97aab2527617e47ff6bdb0d2d756cd59d87305854ff0da70db21",
			rfc6979:  true,
			expected: "ed2b45e5012f71f8713961e52cef31d998fd9123d7db399fc16fa9baa7e4618ea08a1cb3b93d0b85716425d5fb0c650c7fa24dbf0ee1cafb24d43ab8d1523d33",
		},
		{
			name:     "random key 5, sha256(0x05), rfc6979 nonce",
			key:      "915cb9ba4675de06a182088b182abcf79fa8ac989328212c6b866fa3ec2338f9",
			msg:      "05",
			hash:     "e77b9a9ae9e30b0dbdb6f510a264ef9de781501d7b6b92ae89eb059c5ab743db",
			nonce:    "5483b3bd97249c6b105521054124bc767f449909a243466dbb9969e16d6617a4",
			rfc6979:  true,
			expected: "8a76f317a02e95f16d8b50acfb7fe3e7f874af2191466da3f7cdaf0864f283cd3326cfd21edbdb593dc34293ae9b395ca220743920cca6cf24eca117e8bb4330",
		},
		{
			name:     "random key 6, sha256(0x06), rfc6979 nonce",
			key:      "93e9d81d818f08ba1f850c6dfb82256b035b42f7d43c1fe090804fb009aca441",
			msg:      "06",
			hash:     "67586e98fad27da0b9968bc039a1ef34c939b9b8e523a8bef89d478608c5ecf6",
			nonce:    "1cd129fa7b10bf9010d57a8f1e64c90dbcfefce8c07cca2329bd3461abd7d58e",
			rfc6979:  true,
			expected: "9fbdab6bfec294e1ad3d97ef1cc62c6aa580fe4b42cc0336e7677c1a11627f82dc01baae166a321625f9a27f23241afaa30ae21e035c6ac060ae8318d317615c",
		},
		{
			name:     "random key 7, sha256(0x07), rfc6979 nonce",
			key:      "c249bbd5f533672b7dcd514eb1256854783531c2b85fe60bf4ce6ea1f26afc2b",
			msg:      "07",
			hash:     "ca358758f6d27e6cf45272937977a748fd88391db679ceda7dc7bf1f005ee879",
			nonce:    "2fe9c257a73f348628ac0c83a1661a71564d3988427692011e6d3d4f67a589fb",
			rfc6979:  true,
			expected: "8e4ba15db0092985fe310c6863df5d964c08f6b514eaac8a4eb0e629fc6cf415ccec5f049d1d256756bc4eea24a3a27218435c09de6115ca1fc7cabed55dda46",
		},
		{
			name:     "random key 8, sha256(0x08), rfc6979 nonce",
			key:      "ec0be92fcec66cf1f97b5c39f83dfd4ddcad0dad468d3685b5eec556c6290bcc",
			msg:      "08",
			hash:     "beead77994cf573341ec17b58bbf7eb34d2711c993c1d976b128b3188dc1829a",
			nonce:    "de9c1da6a8d955284541694b3e7b327f99706ebab2f8cbc05934cffd6d12ff10",
			rfc6979:  true,
			expected: "b97eb08e39defe8c7764947c984b71c5f8883fabe7aaa28ef0b7643794e4f998e10e00dd4cfb11fcf85c59ef434141bd476b2c10c49b5772c14bcae3fc567dbf",
		},
		{
			name:     "random key 9, sha256(0x09), rfc6979 nonce",
			key:      "6847b071a7cba6a85099b26a9c3e57a964e4990620e1e1c346fecc4472c4d834",
			msg:      "09",
			hash:     "2b4c342f5433ebe591a1da77e013d1b72475562d48578dca8b84bac6651c3cb9",
			nonce:    "3f43590ff9c2028852c81c11483e08aed76b32686a15b3132fcc8bff01bed091",
			rfc6979:  true,
			expected: "ad8f5b18087d91c2a7168861aa738891ae548452f431d1fed864d43ad1b9048cb07c6b9924603e46b83747be359e597aabfcab2e06ddf181882526420988d8cf",
		},
		{
			name:     "random key 10, sha256(0x0a), rfc6979 nonce",
			key:      "b7548540f52fe20c161a0d623097f827608c56023f50442cc00cc50ad674f6b5",
			msg:      "0a",
			hash:     "01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b",
			nonce:    "797bb264572303bac579ed1ec0824f4b3a788128e468c25b6bb4b14ace3ab278",
			rfc6979:  true,
			expected: "2d25d3e9b2b7bc9f8b8228b7e1ca0776b944a67232dff1f9d1494903ce4f2fa74c3ba73a05be46d75851f010f64ca1d492daac66f94eee56799ff6ae1ada76e5",
		},
	}

	for _, test := range tests {
		secKey := hexToModNScalar(test.key)
		msg := hexToBytes(test.msg)
		hash := hexToBytes(test.hash)
		nonce := hexToModNScalar(test.nonce)
		wantSig := hexToBytes(test.expected)

		// Ensure the test data is sane by comparing the provided hashed message
		// and nonce, in the case rfc6979 was used, to their calculated values.
		// These values could just be calculated instead of specified in the
		// test data, but it's nice to have all of the calculated values
		// available in the test data for cross implementation testing and
		// verification.
		calcHash := sha256.Sum256(msg)
		if !bytes.Equal(calcHash[:], hash) {
			t.Errorf("%s: mismatched test hash -- expected: %x, given: %x",
				test.name, calcHash[:], hash)
			continue
		}
		if test.rfc6979 {
			secKeyBytes := hexToBytes(test.key)
			nonceBytes := hexToBytes(test.nonce)
			calcNonce := secp256k1.NonceRFC6979(secKeyBytes, hash,
				rfc6979ExtraDataV0[:], nil, 0)
			calcNonceBytes := calcNonce.Bytes()
			if !bytes.Equal(calcNonceBytes[:], nonceBytes) {
				t.Errorf("%s: mismatched test nonce -- expected: %x, given: %x",
					test.name, calcNonceBytes, nonceBytes)
				continue
			}
		}

		// Sign the hash of the message with the given secret key and nonce.
		gotSig, err := schnorrSign(secKey, nonce, hash)
		if err != nil {
			t.Errorf("%s: unexpected error when signing: %v", test.name, err)
			continue
		}

		// Ensure the generated signature is the expected value.
		gotSigBytes := gotSig.Serialize()
		if !bytes.Equal(gotSigBytes, wantSig) {
			t.Errorf("%s: unexpected signature -- got %x, want %x", test.name,
				gotSigBytes, wantSig)
			continue
		}

		// Ensure the produced signature verifies as well.
		pubKey := secp256k1.NewSecretKey(hexToModNScalar(test.key)).PubKey()
		err = schnorrVerify(gotSig, hash, pubKey)
		if err != nil {
			t.Errorf("%s: signature failed to verify: %v", test.name, err)
			continue
		}
	}
}

// TestSchnorrSignAndVerifyRandom ensures the Schnorr signing and verification
// work as expected for randomly-generated secret keys and messages.  It also
// ensures invalid signatures are not improperly verified by mutating the valid
// signature and changing the message the signature covers.
func TestSchnorrSignAndVerifyRandom(t *testing.T) {
	// Use a unique random seed each test instance and log it if the tests fail.
	seed := time.Now().Unix()
	rng := rand.New(rand.NewSource(seed))
	defer func(t *testing.T, seed int64) {
		if t.Failed() {
			t.Logf("random seed: %d", seed)
		}
	}(t, seed)

	for i := 0; i < 100; i++ {
		// Generate a random secret key.
		var buf [32]byte
		if _, err := rng.Read(buf[:]); err != nil {
			t.Fatalf("failed to read random secret key: %v", err)
		}
		var secKeyScalar secp256k1.ModNScalar
		secKeyScalar.SetBytes(&buf)
		secKey := secp256k1.NewSecretKey(&secKeyScalar)

		// Generate a random hash to sign.
		var hash [32]byte
		if _, err := rng.Read(hash[:]); err != nil {
			t.Fatalf("failed to read random hash: %v", err)
		}

		// Sign the hash with the secret key and then ensure the produced
		// signature is valid for the hash and public key associated with the
		// secret key.
		sig, err := Sign(secKey, hash[:])
		if err != nil {
			t.Fatalf("failed to sign\nsecret key: %x\nhash: %x",
				secKey.Serialize(), hash)
		}
		pubKey := secKey.PubKey()
		if !sig.Verify(hash[:], pubKey) {
			t.Fatalf("failed to verify signature\nsig: %x\nhash: %x\n"+
				"secret key: %x\npublic key: %x", sig.Serialize(), hash,
				secKey.Serialize(), pubKey.SerializeCompressed())
		}

		// Change a random bit in the signature and ensure the bad signature
		// fails to verify the original message.
		goodSigBytes := sig.Serialize()
		badSigBytes := make([]byte, len(goodSigBytes))
		copy(badSigBytes, goodSigBytes)
		randByte := rng.Intn(len(badSigBytes))
		randBit := rng.Intn(7)
		badSigBytes[randByte] ^= 1 << randBit
		badSig, err := ParseSignature(badSigBytes)
		if err != nil {
			t.Fatalf("failed to create bad signature: %v", err)
		}
		if badSig.Verify(hash[:], pubKey) {
			t.Fatalf("verified bad signature\nsig: %x\nhash: %x\n"+
				"secret key: %x\npublic key: %x", badSig.Serialize(), hash,
				secKey.Serialize(), pubKey.SerializeCompressed())
		}

		// Change a random bit in the hash that was originally signed and ensure
		// the original good signature fails to verify the new bad message.
		badHash := make([]byte, len(hash))
		copy(badHash, hash[:])
		randByte = rng.Intn(len(badHash))
		randBit = rng.Intn(7)
		badHash[randByte] ^= 1 << randBit
		if sig.Verify(badHash[:], pubKey) {
			t.Fatalf("verified signature for bad hash\nsig: %x\nhash: %x\n"+
				"pubkey: %x", sig.Serialize(), badHash,
				pubKey.SerializeCompressed())
		}
	}
}

// TestVerifyErrors ensures several error paths in Schnorr verification are
// detected as expected.  When possible, the signatures are otherwise valid with
// the exception of the specific failure to ensure it's robust against things
// like fault attacks.
//
// todo: change this test set to work with sha256 hashes.
func TestVerifyErrors(t *testing.T) {
	tests := []struct {
		name string // test description
		sigR string // hex encoded r component of signature to verify against
		sigS string // hex encoded s component of signature to verify against
		hash string // hex encoded hash of message to verify
		pubX string // hex encoded x component of pubkey to verify against
		pubY string //  hex encoded y component of pubkey to verify against
		err  error  // expected error
	}{{
		// Signature created from secret key 0x01, blake256(0x01020304) || 00.
		// It is otherwise valid.
		name: "hash too long",
		sigR: "4c68976afe187ff0167919ad181cb30f187e2af1c8233b2cbebbbe0fc97fff61",
		sigS: "e77c69035738000caed6ab0ce1eabe5f7e105498f84d0e8982e87ee4da21948e",
		hash: "c301ba9de5d6053caad9f5eb46523f007702add2c62fa39de03146a36b8026b700",
		pubX: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubY: "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		err:  ErrInvalidHashLen,
	}, {
		// Signature created from secret key 0x01, blake256(0x40) and removing
		// the leading zero byte.  It is otherwise valid.
		name: "hash too short",
		sigR: "938de23d0785c7d4775f47bbcadaa2a56447dd98029c8196f2bbed0ab4b8457f",
		sigS: "7de65bf205e14f81e5f75ad2fd80ea715a391f7b51e10fa43f0a1961039b1a6c",
		hash: "0e0f08e2ee912478b77004ec62845b5e01418f03837b76cbdc8b1fb0480322",
		pubX: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubY: "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		err:  ErrInvalidHashLen,
	}, {
		// Signature created from secret key 0x01, blake256(0x01020304) over
		// the secp256r1 curve (note the r1 instead of k1).
		name: "pubkey not on the curve, signature valid for secp256r1 instead",
		sigR: "c6c62660176b3daa90dbf4d7e21d9406ce93895771a16c7c5c91258a9b522174",
		sigS: "f5b5583956a6b30e18ff5e865c77a8c4adf47b147d11ea3822b4de63c9f7b909",
		hash: "c301ba9de5d6053caad9f5eb46523f007702add2c62fa39de03146a36b8026b7",
		pubX: "6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296",
		pubY: "4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5",
		err:  ErrPubKeyNotOnCurve,
	}, {
		// Signature invented since finding a signature with an r value that is
		// exactly the field prime prior to the modular reduction is not
		// calculable without breaking the underlying crypto.
		name: "r == field prime",
		sigR: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f",
		sigS: "e9ae2d0e306497236d4e328dc1a34244045745e87da69d806859348bc2a74525",
		hash: "c301ba9de5d6053caad9f5eb46523f007702add2c62fa39de03146a36b8026b7",
		pubX: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubY: "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		err:  ErrSigRTooBig,
	}, {
		// Likewise, signature invented since finding a signature with an r
		// value that would be valid modulo the field prime and is still 32
		// bytes is not calculable without breaking the underlying crypto.
		name: "r > field prime (prime + 1)",
		sigR: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc30",
		sigS: "e9ae2d0e306497236d4e328dc1a34244045745e87da69d806859348bc2a74525",
		hash: "c301ba9de5d6053caad9f5eb46523f007702add2c62fa39de03146a36b8026b7",
		pubX: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubY: "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		err:  ErrSigRTooBig,
	}, {
		// Signature invented since finding a signature with an s value that is
		// exactly the group order prior to the modular reduction is not
		// calculable without breaking the underlying crypto.
		name: "s == group order",
		sigR: "4c68976afe187ff0167919ad181cb30f187e2af1c8233b2cbebbbe0fc97fff61",
		sigS: "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141",
		hash: "c301ba9de5d6053caad9f5eb46523f007702add2c62fa39de03146a36b8026b7",
		pubX: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubY: "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		err:  ErrSigSTooBig,
	}, {
		// Likewise, signature invented since finding a signature with an s
		// value that would be valid modulo the group order and is still 32
		// bytes is not calculable without breaking the underlying crypto.
		name: "s > group order and still 32 bytes (order + 1)",
		sigR: "4c68976afe187ff0167919ad181cb30f187e2af1c8233b2cbebbbe0fc97fff61",
		sigS: "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364142",
		hash: "c301ba9de5d6053caad9f5eb46523f007702add2c62fa39de03146a36b8026b7",
		pubX: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubY: "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		err:  ErrSigSTooBig,
	}, {
		// Signature created from secret key 0x01, blake256(0x01020304) and
		// manually setting s = -ed.
		//
		// Signature is otherwise invalid too since finding a signature where
		// the two points add to infinity while still having a matching r is not
		// calculable.
		name: "calculated R point at infinity",
		sigR: "4c68976afe187ff0167919ad181cb30f187e2af1c8233b2cbebbbe0fc97fff61",
		sigS: "14cc9e0544dd8fe6baa7c20fd2a141d0ee60114c419377efc850a49bd5c1ed36",
		hash: "c301ba9de5d6053caad9f5eb46523f007702add2c62fa39de03146a36b8026b7",
		pubX: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubY: "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		err:  ErrSigRNotOnCurve,
	}, {
		// Signature created from secret key 0x01, blake256(0x01020304050607).
		// It is otherwise valid.
		name: "odd R",
		sigR: "2c2c71f7bf3e183238b1f20d856e068dc6d37805c8b2d872d0f23d906bc95789",
		sigS: "eb7670ca6ff95c1d5c6785bc72e0781f27c9778758317d82d3053fdbcc9c17b0",
		hash: "ccf8c53a7631aad469d412963d495c729ff219dd2ae9a0c4de4bd1b4c777d49c",
		pubX: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubY: "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		err:  ErrSigRYIsOdd,
	}, {
		// Signature created from secret key 0x01, blake256(0x01020304).  Thus,
		// it is valid for that message.  Attempting to verify wrong message
		// blake256(0x01020307).
		name: "mismatched R",
		sigR: "4c68976afe187ff0167919ad181cb30f187e2af1c8233b2cbebbbe0fc97fff61",
		sigS: "e9ae2d0e306497236d4e328dc1a34244045745e87da69d806859348bc2a74525",
		hash: "d4f9aea8c329f57a81397f0418269a8bd495957ea56ae0af0dfa886fb5977046",
		pubX: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubY: "483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		err:  ErrUnequalRValues,
	}}
	// NOTE: There is no test for e >= group order because it would require
	// finding a preimage that hashes to the value range [n, 2^256) and n is
	// close enough to 2^256 that there is only roughly a 1 in 2^128 chance of
	// a given hash falling in that range.  In other words, it's not feasible
	// to calculate.

	for _, test := range tests {
		// Parse test data into types.
		hash := hexToBytes(test.hash)
		pubX, pubY := hexToFieldVal(test.pubX), hexToFieldVal(test.pubY)
		pubKey := secp256k1.NewPublicKey(pubX, pubY)

		// Create the serialized signature from the bytes and attempt to parse
		// it to ensure the cases where the r and s components exceed the
		// allowed range is caught.
		sig, err := ParseSignature(hexToBytes(test.sigR + test.sigS))
		if err != nil {
			if !errors.Is(err, test.err) {
				t.Errorf("%s: mismatched err -- got %v, want %v", test.name,
					err,
					test.err)
			}

			continue
		}

		// Ensure the expected error is hit.
		err = schnorrVerifyBlake256(sig, hash, pubKey)
		if !errors.Is(err, test.err) {
			t.Errorf("%s: mismatched err -- got %v, want %v", test.name, err,
				test.err)
			continue
		}
	}
}
